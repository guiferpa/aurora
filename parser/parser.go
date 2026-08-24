package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/token"
)

// A Parser turns a chain of tokens into a tree.
//
// One parser serves every source there is: everything a parse needs arrives at Parse, and a
// parser is built from nothing at all. It used to take the tokens at build time, so an
// instance parsed exactly one file and nobody could be handed a parser; only a way of making
// one.
//
// Parse keeps no state of its own between calls, so the same parser may be asked for one tree
// after another.
type Parser interface {
	Parse(in ParseInput) (ast.AST, error)
}

// ParseInput is one source to read, and everything reading it takes.
type ParseInput struct {
	// Filename is the source path. It decides file-scoped rules: assert is only accepted
	// inside *.test.ar.
	Filename string
	Tokens   []token.Token
	// TapeSize is the width in bytes of every value, which the parser needs to refuse a
	// number that does not fit one, text longer than one, and a tape literal with too many
	// items. Zero means the language's default.
	//
	// It is a property of the project the source belongs to, and it arrives here rather than
	// at the build because a parser can outlive it: the language server holds one for a whole
	// session, and answers for files of different projects — and for the same project after
	// its manifest is edited, which has to reach the next keystroke.
	TapeSize int
	// Module is the module this file is, and it decides what the names inside it are called:
	// inside a/b/c, `ident base` is written a/b/c.base, so two modules binding the same word
	// are two different names. Empty is the file somebody asked to run, whose names are
	// written as they were typed — which is every file compiled on its own, the language
	// server and the REPL included.
	Module string
	// Imports is what the modules this file names offer, by specifier: the shapes they
	// declare, and what their scopes said they answer with.
	//
	// It arrives because a shape is resolved while parsing and a shape's name never leaves
	// the file that declared it — so a scope of another module answering with one has to hand
	// the fields over. Empty is a file that imports nothing, or one whose imports promised
	// nothing, and it reads exactly as it did before any of this.
	Imports map[string]ast.Offer
	// Declarations carries what `shape` and `as` declared across parses of the same file.
	// Nil starts empty, which is what compiling a file in one go wants; the REPL passes the
	// same value every line so a shape declared earlier is still known.
	Declarations *Declarations
}

type pr struct {
	filename string
	cursor   int
	tokens   []token.Token
	tapeSize int
	// declarations is what `shape` and `as` leave behind, and nothing more: it turns `p.x`
	// into an index and never reaches the tree, the IR or the binary. It is held by
	// reference so a caller compiling one file across several parses — the REPL — keeps
	// what was declared earlier.
	declarations *Declarations
	// useAllowed says whether a `use` may still be read: it is true at the start of a file
	// and false from the first node that is not one, and inside every body.
	useAllowed bool
	module     string
	// imports is what the modules this file names offer, by specifier.
	imports map[string]ast.Offer
	// references is every qualified name this parse read. It leaves with the tree because
	// only whoever holds the other modules can say whether the name is really there.
	references []ast.Reference
}

// Helper functions to validate node types for tape operations
func isValidTapeTarget(node ast.Node) bool {
	switch node.(type) {
	case ast.TapeBracketExpression, ast.NumberLiteral, ast.IdentifierLiteral,
		ast.PullExpression, ast.PushExpression, ast.HeadExpression, ast.TailExpression:
		return true
	default:
		return false
	}
}

func isValidTapeItem(node ast.Node) bool {
	switch node.(type) {
	case ast.TapeBracketExpression, ast.NumberLiteral, ast.IdentifierLiteral:
		return true
	default:
		return false
	}
}

func (p *pr) ParseCallee(id ast.IdentifierLiteral) (ast.Node, error) {
	params := make([]ast.ParameterLiteral, 0)
	if p.GetLookahead().GetTag().Id != token.O_PAREN {
		return id, nil
	}
	if _, err := p.EatToken(token.O_PAREN); err != nil {
		return nil, err
	}
	for p.GetLookahead().GetTag().Id != token.C_PAREN {
		expr, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		params = append(params, ast.ParameterLiteral{Expression: expr})
		if p.GetLookahead().GetTag().Id == token.C_PAREN {
			break
		}
		if _, err := p.EatToken(token.COMMA); err != nil {
			return nil, err
		}
	}
	if _, err := p.EatToken(token.C_PAREN); err != nil {
		return nil, err
	}
	return ast.CalleeLiteral{Id: id, Params: params}, nil
}

// ParseIdentifier reads a plain name.
func (p *pr) ParseIdentifier() (ast.IdentifierLiteral, error) {
	tok, err := p.EatToken(token.ID)
	if err != nil {
		return ast.IdentifierLiteral{}, err
	}
	return ast.IdentifierLiteral{Value: p.name(string(tok.GetMatch())), Token: tok}, nil
}

func (p *pr) ParseBooleanTrue() (ast.BooleanLiteral, error) {
	tok, err := p.EatToken(token.TRUE)
	if err != nil {
		return ast.BooleanLiteral{}, err
	}
	return ast.BooleanLiteral{Value: byteutil.TrueTape(p.tapeSize), Token: tok}, nil
}

func (p *pr) ParseBooleanFalse() (ast.BooleanLiteral, error) {
	tok, err := p.EatToken(token.FALSE)
	if err != nil {
		return ast.BooleanLiteral{}, err
	}
	return ast.BooleanLiteral{Value: byteutil.FalseTape(p.tapeSize), Token: tok}, nil
}

func (p *pr) ParseNumber() (ast.NumberLiteral, error) {
	tok, err := p.EatToken(token.NUMBER)
	if err != nil {
		return ast.NumberLiteral{}, err
	}

	raw := strings.ReplaceAll(string(tok.GetMatch()), "_", "")

	// Check if it's a hexadecimal number (starts with 0x)
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		if n, err := strconv.ParseUint(raw[2:], 16, 64); err == nil {
			return p.fitInTape(n, tok)
		}
		return ast.NumberLiteral{}, err
	}

	// Parse as decimal
	if n, err := strconv.ParseUint(raw, 10, 64); err == nil {
		return p.fitInTape(n, tok)
	}
	return ast.NumberLiteral{}, err
}

// fitInTape rejects a literal that the configured tape cannot hold, instead of letting it
// be truncated silently at emission.
func (p *pr) fitInTape(v uint64, tok token.Token) (ast.NumberLiteral, error) {
	if !byteutil.FitsInTape(v, p.tapeSize) {
		return ast.NumberLiteral{}, token.NewError(tok, "value %d does not fit in a %d-byte tape (max %d)", v, p.tapeSize, byteutil.MaxTapeValue(p.tapeSize))
	}
	return ast.NumberLiteral{Value: v, Token: tok}, nil
}

// ParseText reads `"text"` into one tape holding its bytes.
//
// The bytes are the text as written, in UTF-8, right aligned like every other value — so
// "a" is the tape holding 97, which is the tape the number 97 is, and "café" is its five
// UTF-8 bytes rather than four characters spread over four tapes.
func (p *pr) ParseText() (ast.TextLiteral, error) {
	tok, err := p.EatToken(token.STRING)
	if err != nil {
		return ast.TextLiteral{}, err
	}
	match := tok.GetMatch()
	if len(match) < 2 {
		return ast.TextLiteral{}, token.NewError(tok, "invalid string literal at line %d, column %d", tok.GetLine(), tok.GetColumn())
	}
	content := match[1 : len(match)-1]

	// A value is a tape, and a tape is tape_size bytes: text that does not fit is rejected
	// where it was written, the same way a number that does not fit is.
	if len(content) > p.tapeSize {
		return ast.TextLiteral{}, token.NewError(tok, "text is %d bytes but a tape holds %d at line %d and column %d",
			len(content), p.tapeSize, tok.GetLine(), tok.GetColumn())
	}

	return ast.TextLiteral{Value: byteutil.PaddingTape(content, p.tapeSize), Token: tok}, nil
}

// ParsePriExpr reads a primary and then whatever binds to it tightest: a field, a shape.
func (p *pr) ParsePriExpr() (ast.Node, error) {
	expr, err := p.parsePrimaryExpr()
	if err != nil {
		return nil, err
	}
	return p.parsePostfix(expr)
}

func (p *pr) parsePrimaryExpr() (ast.Node, error) {
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == token.FEED {
		return p.ParseFeed()
	}
	if lookahead.GetTag().Id == token.O_PAREN {
		if _, err := p.EatToken(token.O_PAREN); err != nil {
			return nil, err
		}
		expr, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.EatToken(token.C_PAREN); err != nil {
			return nil, err
		}
		return expr, nil
	}
	if lookahead.GetTag().Id == token.O_BRK {
		return p.ParseTapeBrk()
	}
	if lookahead.GetTag().Id == token.NUMBER {
		num, err := p.ParseNumber()
		if err != nil {
			return nil, err
		}
		return num, nil
	}
	if lookahead.GetTag().Id == token.STRING {
		text, err := p.ParseText()
		if err != nil {
			return nil, err
		}
		return text, nil
	}
	if lookahead.GetTag().Id == token.TRUE {
		return p.ParseBooleanTrue()
	}
	if lookahead.GetTag().Id == token.FALSE {
		return p.ParseBooleanFalse()
	}
	if lookahead.GetTag().Id == token.O_CUR_BRK {
		return p.ParseBlockExpr()
	}
	if lookahead.GetTag().Id == token.IDENT {
		return p.ParseIdent()
	}
	id, err := p.ParseIdentifier()
	if err != nil {
		return nil, err
	}
	// An alias names a module, and a module is not a value: nothing loads under it. Reaching
	// something inside is the only thing it is for, and that is a dot away.
	if specifier, isModule := p.declarations.Modules[typed(id)]; isModule {
		if p.GetLookahead() != nil && p.GetLookahead().GetTag().Id == token.DOT {
			return id, nil
		}
		return nil, token.NewError(id.Token, "%s is the module %s at line %d and column %d: reach something inside it with %s.name",
			typed(id), specifier, id.Token.GetLine(), id.Token.GetColumn(), typed(id))
	}
	// A shape's name never reaches the instructions and never leaves the file, so it is
	// looked up as it was typed rather than as the module writes it.
	if _, declared := p.declarations.Shapes[typed(id)]; declared {
		if p.GetLookahead() != nil && p.GetLookahead().GetTag().Id == token.O_CUR_BRK {
			return p.ParseShapeLiteral(id)
		}
		// A shape is a declaration, not a value: there is nothing to load under its name.
		return nil, token.NewError(id.Token, "%s is a shape at line %d and column %d: build a value with %s{...}",
			typed(id), id.Token.GetLine(), id.Token.GetColumn(), typed(id))
	}
	if p.GetLookahead() != nil && p.GetLookahead().GetTag().Id == token.O_PAREN {
		return p.ParseCallee(id)
	}
	return id, nil
}

func (p *pr) ParseTapeBrk() (ast.Node, error) {
	at, err := p.EatToken(token.O_BRK)
	if err != nil {
		return nil, err
	}
	items := make([]ast.Node, 0)
	for p.GetLookahead().GetTag().Id != token.C_BRK {
		expr, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}

		// Validate: if item is a number literal, it must be between 0 and 255
		// (since tapes store values as direct bytes)
		if numNode, ok := expr.(ast.NumberLiteral); ok {
			if numNode.Value > byteutil.MAX_BYTES {
				return nil, fmt.Errorf("tape values must be between 0 and %d, got %d", byteutil.MAX_BYTES, numNode.Value)
			}
		}

		items = append(items, expr)
		if p.GetLookahead().GetTag().Id == token.C_BRK {
			break
		}
		if _, err := p.EatToken(token.COMMA); err != nil {
			return nil, err
		}
	}
	closing, err := p.EatToken(token.C_BRK)
	if err != nil {
		return nil, err
	}
	if len(items) > p.tapeSize {
		return nil, token.NewError(closing, "tape literal has %d values but a tape holds %d bytes", len(items), p.tapeSize)
	}
	return ast.TapeBracketExpression{Items: items, Token: at}, nil
}

func (p *pr) ParsePull() (ast.Node, error) {
	at, err := p.EatToken(token.PULL)
	if err != nil {
		return nil, err
	}

	target, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	if !isValidTapeTarget(target) {
		return nil, errors.New("it is not a valid append target")
	}

	expr, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	if !isValidTapeItem(expr) {
		return nil, errors.New("it is not a valid append item")
	}
	return ast.PullExpression{Target: target, Item: expr, Token: at}, nil
}

func (p *pr) ParseHead() (ast.Node, error) {
	at, err := p.EatToken(token.HEAD)
	if err != nil {
		return nil, err
	}
	expr, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	if !isValidTapeTarget(expr) {
		return nil, errors.New("it is not a valid head target")
	}

	length, err := p.ParseNumber()
	if err != nil {
		return nil, err
	}
	return ast.HeadExpression{Expression: expr, Length: length.Value, Token: at}, nil
}

func (p *pr) ParseTail() (ast.Node, error) {
	at, err := p.EatToken(token.TAIL)
	if err != nil {
		return nil, err
	}
	expr, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	if !isValidTapeTarget(expr) {
		return nil, errors.New("it is not a valid tail target")
	}

	length, err := p.ParseNumber()
	if err != nil {
		return nil, err
	}
	return ast.TailExpression{Expression: expr, Length: length.Value, Token: at}, nil
}

func (p *pr) ParsePush() (ast.Node, error) {
	at, err := p.EatToken(token.PUSH)
	if err != nil {
		return nil, err
	}

	target, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	if !isValidTapeTarget(target) {
		return nil, errors.New("it is not a valid push target")
	}

	expr, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	if !isValidTapeItem(expr) {
		return nil, errors.New("it is not a valid push item")
	}
	return ast.PushExpression{Target: target, Item: expr, Token: at}, nil
}

func (p *pr) ParseUnaExpr() (ast.Node, error) {
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == token.SUB {
		op, err := p.EatToken(token.SUB)
		if err != nil {
			return nil, err
		}
		expr, err := p.ParsePriExpr()
		if err != nil {
			return nil, err
		}
		return ast.UnaryExpression{Expression: expr, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	return p.ParsePriExpr()
}

func (p *pr) ParseExpoExpr() (ast.Node, error) {
	left, err := p.ParseUnaExpr()
	if err != nil {
		return nil, err
	}

	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == token.EXPO {
		op, err := p.EatToken(token.EXPO)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseExpoExpr()
		if err != nil {
			return nil, err
		}
		return ast.BinaryExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}

	return left, nil
}

// ParseMultExpr parses multiplicative expressions left-associatively: a / b / c => (a / b) / c,
// a * b * c => (a * b) * c.
//
// It used to recurse into itself on the right, which grouped the other way: `20 / 5 / 2` read
// as 20 / (5 / 2) and answered 10 instead of 2, and `4 / 2 * 6` read as 4 / (2 * 6) and
// answered 0 instead of 12. Multiplication hid it — it is associative, so only a chain mixing
// it with division gives a different answer.
func (p *pr) ParseMultExpr() (ast.Node, error) {
	left, err := p.ParseExpoExpr()
	if err != nil {
		return nil, err
	}
	for {
		lookahead := p.GetLookahead()
		if lookahead.GetTag().Id == token.MULT {
			op, err := p.EatToken(token.MULT)
			if err != nil {
				return nil, err
			}
			right, err := p.ParseExpoExpr()
			if err != nil {
				return nil, err
			}
			left = ast.BinaryExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}
			continue
		}
		if lookahead.GetTag().Id == token.DIV {
			op, err := p.EatToken(token.DIV)
			if err != nil {
				return nil, err
			}
			right, err := p.ParseExpoExpr()
			if err != nil {
				return nil, err
			}
			left = ast.BinaryExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}
			continue
		}
		break
	}
	return left, nil
}

// ParseAddExpr parses additive expressions left-associatively: a - b - c => (a - b) - c, a + b + c => (a + b) + c.
func (p *pr) ParseAddExpr() (ast.Node, error) {
	left, err := p.ParseMultExpr()
	if err != nil {
		return nil, err
	}
	for {
		lookahead := p.GetLookahead()
		if lookahead.GetTag().Id == token.SUM {
			op, err := p.EatToken(token.SUM)
			if err != nil {
				return nil, err
			}
			right, err := p.ParseMultExpr()
			if err != nil {
				return nil, err
			}
			left = ast.BinaryExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}
			continue
		}
		if lookahead.GetTag().Id == token.SUB {
			op, err := p.EatToken(token.SUB)
			if err != nil {
				return nil, err
			}
			right, err := p.ParseMultExpr()
			if err != nil {
				return nil, err
			}
			left = ast.BinaryExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}
			continue
		}
		break
	}
	return left, nil
}

func (p *pr) ParseRelExpr() (ast.Node, error) {
	left, err := p.ParseAddExpr()
	if err != nil {
		return nil, err
	}
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == token.EQUALS {
		op, err := p.EatToken(token.EQUALS)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseRelExpr()
		if err != nil {
			return nil, err
		}
		return ast.RelativeExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	if lookahead.GetTag().Id == token.DIFFERENT {
		op, err := p.EatToken(token.DIFFERENT)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseRelExpr()
		if err != nil {
			return nil, err
		}
		return ast.RelativeExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	if lookahead.GetTag().Id == token.BIGGER {
		op, err := p.EatToken(token.BIGGER)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseRelExpr()
		if err != nil {
			return nil, err
		}
		return ast.RelativeExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	if lookahead.GetTag().Id == token.SMALLER {
		op, err := p.EatToken(token.SMALLER)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseRelExpr()
		if err != nil {
			return nil, err
		}
		return ast.RelativeExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	return left, nil
}

// ParseBoolExpr parses `or`, which binds loosest of all, left-associatively.
//
// `and` and `or` used to share this one level and recurse to the right, so `a and b or c`
// read as `a and (b or c)`: `false and true or true` answered false where every language
// that gives `and` the tighter binding answers true. Splitting them into two levels is what
// makes `and` bind tighter, and the loop is what groups a chain to the left.
func (p *pr) ParseBoolExpr() (ast.Node, error) {
	left, err := p.ParseAndExpr()
	if err != nil {
		return nil, err
	}
	for p.GetLookahead().GetTag().Id == token.OR {
		op, err := p.EatToken(token.OR)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseAndExpr()
		if err != nil {
			return nil, err
		}
		left = ast.BooleanExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}
	}
	return left, nil
}

// ParseAndExpr parses `and`, which binds tighter than `or` and looser than a comparison.
func (p *pr) ParseAndExpr() (ast.Node, error) {
	left, err := p.ParseRelExpr()
	if err != nil {
		return nil, err
	}
	for p.GetLookahead().GetTag().Id == token.AND {
		op, err := p.EatToken(token.AND)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseRelExpr()
		if err != nil {
			return nil, err
		}
		left = ast.BooleanExpression{Left: left, Right: right, Operation: ast.OperationLiteral{Value: string(op.GetMatch()), Token: op}}
	}
	return left, nil
}

func (p *pr) ParseBranchItem() (ast.Node, error) {
	expr, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}

	if p.GetLookahead().GetTag().Id == token.SEMICOLON {
		if _, err := p.EatToken(token.SEMICOLON); err != nil {
			return nil, err
		}
		return expr, nil
	}

	_, isBoolean := expr.(ast.BooleanExpression)
	_, isRel := expr.(ast.RelativeExpression)
	_, isLiteralBool := expr.(ast.BooleanLiteral)
	_, isId := expr.(ast.IdentifierLiteral)
	if !isBoolean && !isRel && !isLiteralBool && !isId {
		return nil, errors.New("branch must have boolean expression as test")
	}

	if _, err := p.EatToken(token.COLON); err != nil {
		return nil, err
	}

	body, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}

	if _, err := p.EatToken(token.COMMA); err != nil {
		return nil, err
	}

	euzeb, err := p.ParseBranchItem()
	if err != nil {
		return nil, err
	}

	return ast.IfExpression{
		Test: expr,
		Body: []ast.Node{body},
		Else: &ast.ElseExpression{Body: []ast.Node{euzeb}},
	}, nil
}

func (p *pr) ParseBranch() (ast.Node, error) {
	if _, err := p.EatToken(token.BRANCH); err != nil {
		return nil, err
	}

	if _, err := p.EatToken(token.O_CUR_BRK); err != nil {
		return nil, err
	}

	item, err := p.ParseBranchItem()
	if err != nil {
		return nil, err
	}

	if _, err := p.EatToken(token.C_CUR_BRK); err != nil {
		return nil, err
	}

	return item, nil
}

// ParseBlockExpr reads a block where an expression is wanted.
func (p *pr) ParseBlockExpr() (ast.Node, error) {
	return p.parseBlock()
}

// parseBlock reads `{ ... }` and the promise that may follow it.
//
// A deferred scope is a block with a word in front of it, so it comes through here too — which
// is what gives it the promise, and what keeps the braces from being read in two places.
func (p *pr) parseBlock() (ast.BlockExpression, error) {
	if _, err := p.EatToken(token.O_CUR_BRK); err != nil {
		return ast.BlockExpression{}, err
	}
	exprs, err := p.ParseExprs(token.TagCCurBrk)
	if err != nil {
		return ast.BlockExpression{}, err
	}
	closing, err := p.EatToken(token.C_CUR_BRK)
	if err != nil {
		return ast.BlockExpression{}, err
	}

	// The promise is read here, which is the one place it is written. The body of an if is
	// read straight rather than through here, so it never takes one.
	promised, err := p.parseReturns(exprs, closing)
	if err != nil {
		return ast.BlockExpression{}, err
	}

	return ast.BlockExpression{Body: exprs, Returns: promised}, nil
}

func (p *pr) ParseDefer() (ast.Node, error) {
	if _, err := p.EatToken(token.DEFER); err != nil {
		return nil, err
	}
	block, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return ast.DeferExpression{Block: block}, nil
}

func (p *pr) ParseIf() (ast.Node, error) {
	at, err := p.EatToken(token.IF)
	if err != nil {
		return nil, err
	}
	test, err := p.ParseBoolExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(token.O_CUR_BRK); err != nil {
		return nil, err
	}
	body, err := p.ParseExprs(token.TagCCurBrk)
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(token.C_CUR_BRK); err != nil {
		return nil, err
	}
	if p.GetLookahead().GetTag().Id == token.ELSE {
		euze, err := p.ParseElse()
		return ast.IfExpression{Test: test, Body: body, Else: euze, Token: at}, err
	}
	return ast.IfExpression{Test: test, Body: body, Else: nil, Token: at}, nil
}

func (p *pr) ParseElse() (*ast.ElseExpression, error) {
	if _, err := p.EatToken(token.ELSE); err != nil {
		return nil, err
	}
	if _, err := p.EatToken(token.O_CUR_BRK); err != nil {
		return nil, err
	}
	body, err := p.ParseExprs(token.TagCCurBrk)
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(token.C_CUR_BRK); err != nil {
		return nil, err
	}
	return &ast.ElseExpression{Body: body}, nil
}

func (p *pr) ParseIdent() (ast.Node, error) {
	if _, err := p.EatToken(token.IDENT); err != nil {
		return nil, err
	}
	id, err := p.EatToken(token.ID)
	if len(id.GetMatch()) == 0 {
		return nil, token.NewError(id, "missing identifier name at line: %d, column %d", id.GetLine(), id.GetColumn())
	}
	if err != nil {
		return nil, err
	}
	// An alias is a name in this file's scope and follows the rules of every other one, so
	// binding over it is a redeclaration. It can only happen in this order: a use is read
	// before anything else in the file.
	if specifier, ok := p.declarations.Modules[string(id.GetMatch())]; ok {
		return nil, token.NewError(id, "%s is already the alias of %s at line %d and column %d",
			id.GetMatch(), specifier, id.GetLine(), id.GetColumn())
	}
	if _, err := p.EatToken(token.ASSIGN); err != nil {
		return nil, err
	}
	expr, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	// Binding a value with a shape carries the shape to the name, so `p.x` reads after
	// `ident p = feed(0) as Point;`.
	name := p.name(string(id.GetMatch()))
	if shape := p.shapeOf(expr); shape != "" {
		p.declarations.Reads[name] = shape
	}
	// A scope that promised is not itself a shape — its value is an index — so what is
	// written down is what calling it answers with.
	if scope, ok := expr.(ast.DeferExpression); ok && scope.Block.Returns != "" {
		p.declarations.Returns[name] = scope.Block.Returns
	}
	return ast.IdentLiteral{Id: name, Token: id, Value: expr}, nil
}

func (p *pr) ParseExpr() (ast.Node, error) {
	lookahead := p.GetLookahead()
	if lookahead == nil {
		return nil, fmt.Errorf("unexpected end of input")
	}
	if format, ok := printFormats[lookahead.GetTag().Id]; ok {
		return p.ParsePrint(lookahead.GetTag().Id, format)
	}
	if lookahead.GetTag().Id == token.ASSERT {
		return p.ParseAssert()
	}
	if lookahead.GetTag().Id == token.USE {
		return p.ParseUse()
	}
	if lookahead.GetTag().Id == token.SHAPE {
		return p.ParseShape()
	}
	if lookahead.GetTag().Id == token.O_CUR_BRK {
		return p.ParseBlockExpr()
	}
	if lookahead.GetTag().Id == token.IF {
		return p.ParseIf()
	}
	if lookahead.GetTag().Id == token.BRANCH {
		return p.ParseBranch()
	}
	if lookahead.GetTag().Id == token.DEFER {
		return p.ParseDefer()
	}
	if lookahead.GetTag().Id == token.IDENT {
		return p.ParseIdent()
	}
	if lookahead.GetTag().Id == token.PULL {
		return p.ParsePull()
	}
	if lookahead.GetTag().Id == token.HEAD {
		return p.ParseHead()
	}
	if lookahead.GetTag().Id == token.TAIL {
		return p.ParseTail()
	}
	if lookahead.GetTag().Id == token.PUSH {
		return p.ParsePush()
	}
	return p.ParseBoolExpr()
}

// printFormats maps each print builtin to the reading it asks for.
var printFormats = map[string]ast.PrintFormat{
	token.PRINTB: ast.PrintBytes,
	token.PRINTC: ast.PrintChars,
	token.PRINTD: ast.PrintDecimal,
}

func (p *pr) ParsePrint(tokenName string, format ast.PrintFormat) (ast.Node, error) {
	at, err := p.EatToken(tokenName)
	if err != nil {
		return nil, err
	}
	expr, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	return ast.PrintStatement{Format: format, Param: expr, Token: at}, nil
}

func (p *pr) ParseAssert() (ast.Node, error) {
	// Validate that assert can only be used in .test.ar files
	if !strings.HasSuffix(p.filename, ".test.ar") {
		lookahead := p.GetLookahead()
		return nil, token.NewError(lookahead, "assert can only be used in .test.ar files (at line %d, column %d)", lookahead.GetLine(), lookahead.GetColumn())
	}

	t, err := p.EatToken(token.ASSERT)
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(token.O_PAREN); err != nil {
		return nil, err
	}
	condition, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(token.COMMA); err != nil {
		return nil, err
	}
	// The message is a literal rather than an expression: it is text for whoever reads the
	// result, and nothing in the language builds text anyway.
	message := p.GetLookahead()
	if message == nil || message.GetTag().Id != token.STRING {
		return nil, token.NewError(message, "assert needs a message written as text at line %d and column %d",
			t.GetLine(), t.GetColumn())
	}
	if _, err := p.EatToken(token.STRING); err != nil {
		return nil, err
	}
	quoted := message.GetMatch()

	if _, err := p.EatToken(token.C_PAREN); err != nil {
		return nil, err
	}

	return ast.AssertStatement{
		Condition: condition,
		Message:   string(quoted[1 : len(quoted)-1]),
		Token:     t,
	}, nil
}

// ParseFeed parses the builtin "feed": feed(index), or feed index without parentheses.
func (p *pr) ParseFeed() (ast.Node, error) {
	if _, err := p.EatToken(token.FEED); err != nil {
		return nil, err
	}
	if p.GetLookahead().GetTag().Id == token.O_PAREN {
		if _, err := p.EatToken(token.O_PAREN); err != nil {
			return nil, err
		}
		nth, err := p.ParseNumber()
		if err != nil {
			return nil, err
		}
		if _, err := p.EatToken(token.C_PAREN); err != nil {
			return nil, err
		}
		return ast.FeedExpression{Nth: nth}, nil
	}
	nth, err := p.ParseNumber()
	if err != nil {
		return nil, err
	}
	return ast.FeedExpression{Nth: nth}, nil
}

func (p *pr) ParseExprs(t token.Tag) ([]ast.Node, error) {
	// Anything read until a token other than the end of the file is a body, and a body is
	// not the top of a file.
	if t.Id != token.EOF {
		p.useAllowed = false
	}

	exprs := make([]ast.Node, 0)
	for {
		lookahead := p.GetLookahead()
		if lookahead == nil || lookahead.GetTag().Id == t.Id {
			break
		}
		expr, err := p.ParseExpr()
		if err != nil {
			return exprs, err
		}
		if _, err := p.EatToken(token.SEMICOLON); err != nil {
			return exprs, err
		}
		if _, isUse := expr.(ast.UseDeclaration); !isUse {
			p.useAllowed = false
		}
		exprs = append(exprs, expr)
	}
	return exprs, nil
}

func (p *pr) GetLookahead() token.Token {
	if p.cursor >= len(p.tokens) {
		return nil
	}
	return p.tokens[p.cursor]
}

func (p *pr) EatToken(tokenId string) (token.Token, error) {
	currtok := p.GetLookahead()

	if currtok == nil {
		return nil, nil
	}

	if tokenId != currtok.GetTag().Id {
		return currtok, token.NewError(currtok, "unexpected token %s at line %d and column %d", currtok.GetMatch(), currtok.GetLine(), currtok.GetColumn())
	}

	p.cursor++

	return currtok, nil
}

// Parse reads one source and answers with its tree.
//
// The receiver is a value, so what a parse needs — where it is in the chain, which tokens,
// which file — is written on a copy and the parser itself is never touched. Two calls cannot
// tread on each other, which is what lets a host hold one.
func (p pr) Parse(in ParseInput) (ast.AST, error) {
	p.filename = in.Filename
	p.tokens = in.Tokens
	p.tapeSize = byteutil.TapeSize(in.TapeSize)
	p.cursor = 0
	p.useAllowed = true
	p.module = in.Module
	p.imports = in.Imports
	p.references = nil
	p.declarations = in.Declarations
	if p.declarations == nil {
		p.declarations = NewDeclarations()
	}

	nodes, err := (&p).ParseExprs(token.TagEOF)
	if err != nil {
		return ast.AST{}, err
	}

	return ast.AST{
		Filename:   p.filename,
		Nodes:      nodes,
		References: p.references,
		Promises:   p.promises(nodes),
		Shapes:     p.shapes(nodes),
	}, nil
}

// New builds a parser. It takes nothing: a parser is the same whatever it is asked to read,
// and everything a parse needs arrives at Parse.
func New() Parser {
	return pr{}
}
