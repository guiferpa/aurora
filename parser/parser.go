package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/lexer"
)

type Parser interface {
	GetLookahead() lexer.Token
	EatToken(tokenId string) (lexer.Token, error)
	Parse() (AST, error)
}

type pr struct {
	filename string
	cursor   int
	tokens   []lexer.Token
	logger   *Logger
	tapeSize int
}

// Helper functions to validate node types for tape operations
func isValidTapeTarget(node Node) bool {
	switch node.(type) {
	case TapeBracketExpression, NumberLiteral, IdentifierLiteral,
		PullExpression, PushExpression, HeadExpression, TailExpression:
		return true
	default:
		return false
	}
}

func isValidTapeItem(node Node) bool {
	switch node.(type) {
	case TapeBracketExpression, NumberLiteral, IdentifierLiteral:
		return true
	default:
		return false
	}
}

func (p *pr) ParseCallee(id IdentifierLiteral) (Node, error) {
	params := make([]ParameterLiteral, 0)
	if p.GetLookahead().GetTag().Id != lexer.O_PAREN {
		return id, nil
	}
	if _, err := p.EatToken(lexer.O_PAREN); err != nil {
		return nil, err
	}
	for p.GetLookahead().GetTag().Id != lexer.C_PAREN {
		expr, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		params = append(params, ParameterLiteral{expr})
		if p.GetLookahead().GetTag().Id == lexer.C_PAREN {
			break
		}
		if _, err := p.EatToken(lexer.COMMA); err != nil {
			return nil, err
		}
	}
	if _, err := p.EatToken(lexer.C_PAREN); err != nil {
		return nil, err
	}
	return CalleeLiteral{id, params}, nil
}

// ParseIdentifier reads a plain name.
func (p *pr) ParseIdentifier() (IdentifierLiteral, error) {
	tok, err := p.EatToken(lexer.ID)
	if err != nil {
		return IdentifierLiteral{}, err
	}
	return IdentifierLiteral{Value: string(tok.GetMatch()), Token: tok}, nil
}

func (p *pr) ParseBooleanTrue() (BooleanLiteral, error) {
	tok, err := p.EatToken(lexer.TRUE)
	if err != nil {
		return BooleanLiteral{}, err
	}
	return BooleanLiteral{byteutil.TrueTape(p.tapeSize), tok}, nil
}

func (p *pr) ParseBooleanFalse() (BooleanLiteral, error) {
	tok, err := p.EatToken(lexer.FALSE)
	if err != nil {
		return BooleanLiteral{}, err
	}
	return BooleanLiteral{byteutil.FalseTape(p.tapeSize), tok}, nil
}

func (p *pr) ParseNumber() (NumberLiteral, error) {
	tok, err := p.EatToken(lexer.NUMBER)
	if err != nil {
		return NumberLiteral{}, err
	}

	raw := strings.ReplaceAll(string(tok.GetMatch()), "_", "")

	// Check if it's a hexadecimal number (starts with 0x)
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		if n, err := strconv.ParseUint(raw[2:], 16, 64); err == nil {
			return p.fitInTape(n, tok)
		}
		return NumberLiteral{}, err
	}

	// Parse as decimal
	if n, err := strconv.ParseUint(raw, 10, 64); err == nil {
		return p.fitInTape(n, tok)
	}
	return NumberLiteral{}, err
}

// fitInTape rejects a literal that the configured tape cannot hold, instead of letting it
// be truncated silently at emission.
func (p *pr) fitInTape(v uint64, tok lexer.Token) (NumberLiteral, error) {
	if !byteutil.FitsInTape(v, p.tapeSize) {
		return NumberLiteral{}, lexer.NewError(tok, "value %d does not fit in a %d-byte tape (max %d)", v, p.tapeSize, byteutil.MaxTapeValue(p.tapeSize))
	}
	return NumberLiteral{v, tok}, nil
}

func (p *pr) ParseReel() (ReelLiteral, error) {
	tok, err := p.EatToken(lexer.STRING)
	if err != nil {
		return ReelLiteral{}, err
	}
	// Remove surrounding quotes and get the string content
	match := tok.GetMatch()
	if len(match) < 2 {
		return ReelLiteral{}, lexer.NewError(tok, "invalid string literal at line %d, column %d", tok.GetLine(), tok.GetColumn())
	}
	// Remove first and last character (quotes)
	content := match[1 : len(match)-1]

	// A reel is a run of tapes, one per character, and each tape holds the character's
	// number. Ranging over the string rather than the bytes is what keeps a character
	// outside ASCII whole: "café" is four characters, not five bytes.
	reel := make([][]byte, 0, len(content))
	for _, char := range string(content) {
		tape := byteutil.PaddingTape(byteutil.FromUint64(uint64(char)), p.tapeSize)
		reel = append(reel, tape)
	}

	// If empty string, create a reel with one empty tape
	if len(reel) == 0 {
		reel = append(reel, byteutil.FalseTape(p.tapeSize))
	}

	return ReelLiteral{reel, tok}, nil
}

func (p *pr) ParsePriExpr() (Node, error) {
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.FEED {
		return p.ParseFeed()
	}
	if lookahead.GetTag().Id == lexer.O_PAREN {
		if _, err := p.EatToken(lexer.O_PAREN); err != nil {
			return nil, err
		}
		expr, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.EatToken(lexer.C_PAREN); err != nil {
			return nil, err
		}
		return expr, nil
	}
	if lookahead.GetTag().Id == lexer.O_BRK {
		return p.ParseTapeBrk()
	}
	if lookahead.GetTag().Id == lexer.NUMBER {
		num, err := p.ParseNumber()
		if err != nil {
			return nil, err
		}
		return num, nil
	}
	if lookahead.GetTag().Id == lexer.STRING {
		reel, err := p.ParseReel()
		if err != nil {
			return nil, err
		}
		return reel, nil
	}
	if lookahead.GetTag().Id == lexer.TRUE {
		return p.ParseBooleanTrue()
	}
	if lookahead.GetTag().Id == lexer.FALSE {
		return p.ParseBooleanFalse()
	}
	if lookahead.GetTag().Id == lexer.O_CUR_BRK {
		return p.ParseBlockExpr()
	}
	if lookahead.GetTag().Id == lexer.IDENT {
		return p.ParseIdent()
	}
	id, err := p.ParseIdentifier()
	if err != nil {
		return nil, err
	}
	if p.GetLookahead() != nil && p.GetLookahead().GetTag().Id == lexer.O_PAREN {
		return p.ParseCallee(id)
	}
	return id, nil
}

func (p *pr) ParseTapeBrk() (Node, error) {
	if _, err := p.EatToken(lexer.O_BRK); err != nil {
		return nil, err
	}
	items := make([]Node, 0)
	for p.GetLookahead().GetTag().Id != lexer.C_BRK {
		expr, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}

		// Validate: if item is a number literal, it must be between 0 and 255
		// (since tapes store values as direct bytes)
		if numNode, ok := expr.(NumberLiteral); ok {
			if numNode.Value > byteutil.MAX_BYTES {
				return nil, fmt.Errorf("tape values must be between 0 and %d, got %d", byteutil.MAX_BYTES, numNode.Value)
			}
		}

		items = append(items, expr)
		if p.GetLookahead().GetTag().Id == lexer.C_BRK {
			break
		}
		if _, err := p.EatToken(lexer.COMMA); err != nil {
			return nil, err
		}
	}
	closing, err := p.EatToken(lexer.C_BRK)
	if err != nil {
		return nil, err
	}
	if len(items) > p.tapeSize {
		return nil, lexer.NewError(closing, "tape literal has %d values but a tape holds %d bytes", len(items), p.tapeSize)
	}
	return TapeBracketExpression{Items: items}, nil
}

func (p *pr) ParsePull() (Node, error) {
	if _, err := p.EatToken(lexer.PULL); err != nil {
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
	return PullExpression{Target: target, Item: expr}, nil
}

func (p *pr) ParseHead() (Node, error) {
	if _, err := p.EatToken(lexer.HEAD); err != nil {
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
	return HeadExpression{Expression: expr, Length: length.Value}, nil
}

func (p *pr) ParseTail() (Node, error) {
	if _, err := p.EatToken(lexer.TAIL); err != nil {
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
	return TailExpression{Expression: expr, Length: length.Value}, nil
}

func (p *pr) ParsePush() (Node, error) {
	if _, err := p.EatToken(lexer.PUSH); err != nil {
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
	return PushExpression{Target: target, Item: expr}, nil
}

func (p *pr) ParseUnaExpr() (Node, error) {
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.SUB {
		op, err := p.EatToken(lexer.SUB)
		if err != nil {
			return nil, err
		}
		expr, err := p.ParsePriExpr()
		if err != nil {
			return nil, err
		}
		return UnaryExpression{expr, OperationLiteral{string(op.GetMatch()), op}}, nil
	}
	return p.ParsePriExpr()
}

func (p *pr) ParseExpoExpr() (Node, error) {
	left, err := p.ParseUnaExpr()
	if err != nil {
		return nil, err
	}

	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.EXPO {
		op, err := p.EatToken(lexer.EXPO)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseExpoExpr()
		if err != nil {
			return nil, err
		}
		return BinaryExpression{left, right, OperationLiteral{string(op.GetMatch()), op}}, nil
	}

	return left, nil
}

func (p *pr) ParseMultExpr() (Node, error) {
	left, err := p.ParseExpoExpr()
	if err != nil {
		return nil, err
	}

	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.MULT {
		op, err := p.EatToken(lexer.MULT)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseMultExpr()
		if err != nil {
			return nil, err
		}
		return BinaryExpression{left, right, OperationLiteral{string(op.GetMatch()), op}}, nil
	}
	if lookahead.GetTag().Id == lexer.DIV {
		op, err := p.EatToken(lexer.DIV)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseMultExpr()
		if err != nil {
			return nil, err
		}
		return BinaryExpression{left, right, OperationLiteral{string(op.GetMatch()), op}}, nil
	}

	return left, nil
}

// ParseAddExpr parses additive expressions left-associatively: a - b - c => (a - b) - c, a + b + c => (a + b) + c.
func (p *pr) ParseAddExpr() (Node, error) {
	left, err := p.ParseMultExpr()
	if err != nil {
		return nil, err
	}
	for {
		lookahead := p.GetLookahead()
		if lookahead.GetTag().Id == lexer.SUM {
			op, err := p.EatToken(lexer.SUM)
			if err != nil {
				return nil, err
			}
			right, err := p.ParseMultExpr()
			if err != nil {
				return nil, err
			}
			left = BinaryExpression{left, right, OperationLiteral{string(op.GetMatch()), op}}
			continue
		}
		if lookahead.GetTag().Id == lexer.SUB {
			op, err := p.EatToken(lexer.SUB)
			if err != nil {
				return nil, err
			}
			right, err := p.ParseMultExpr()
			if err != nil {
				return nil, err
			}
			left = BinaryExpression{left, right, OperationLiteral{string(op.GetMatch()), op}}
			continue
		}
		break
	}
	return left, nil
}

func (p *pr) ParseRelExpr() (Node, error) {
	left, err := p.ParseAddExpr()
	if err != nil {
		return nil, err
	}
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.EQUALS {
		op, err := p.EatToken(lexer.EQUALS)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseRelExpr()
		if err != nil {
			return nil, err
		}
		return RelativeExpression{left, right, OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	if lookahead.GetTag().Id == lexer.DIFFERENT {
		op, err := p.EatToken(lexer.DIFFERENT)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseRelExpr()
		if err != nil {
			return nil, err
		}
		return RelativeExpression{left, right, OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	if lookahead.GetTag().Id == lexer.BIGGER {
		op, err := p.EatToken(lexer.BIGGER)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseRelExpr()
		if err != nil {
			return nil, err
		}
		return RelativeExpression{left, right, OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	if lookahead.GetTag().Id == lexer.SMALLER {
		op, err := p.EatToken(lexer.SMALLER)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseRelExpr()
		if err != nil {
			return nil, err
		}
		return RelativeExpression{left, right, OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	return left, nil
}

func (p *pr) ParseBoolExpr() (Node, error) {
	left, err := p.ParseRelExpr()
	if err != nil {
		return nil, err
	}
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.OR {
		op, err := p.EatToken(lexer.OR)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseBoolExpr()
		if err != nil {
			return nil, err
		}
		return BooleanExpression{left, right, OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	if lookahead.GetTag().Id == lexer.AND {
		op, err := p.EatToken(lexer.AND)
		if err != nil {
			return nil, err
		}
		right, err := p.ParseBoolExpr()
		if err != nil {
			return nil, err
		}
		return BooleanExpression{left, right, OperationLiteral{Value: string(op.GetMatch()), Token: op}}, nil
	}
	return left, nil
}

func (p *pr) ParseBranchItem() (Node, error) {
	expr, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}

	if p.GetLookahead().GetTag().Id == lexer.SEMICOLON {
		if _, err := p.EatToken(lexer.SEMICOLON); err != nil {
			return nil, err
		}
		return expr, nil
	}

	_, isBoolean := expr.(BooleanExpression)
	_, isRel := expr.(RelativeExpression)
	_, isLiteralBool := expr.(BooleanLiteral)
	_, isId := expr.(IdentifierLiteral)
	if !isBoolean && !isRel && !isLiteralBool && !isId {
		return nil, errors.New("branch must have boolean expression as test")
	}

	if _, err := p.EatToken(lexer.COLON); err != nil {
		return nil, err
	}

	body, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}

	if _, err := p.EatToken(lexer.COMMA); err != nil {
		return nil, err
	}

	euzeb, err := p.ParseBranchItem()
	if err != nil {
		return nil, err
	}

	return IfExpression{
		Test: expr,
		Body: []Node{body},
		Else: &ElseExpression{Body: []Node{euzeb}},
	}, nil
}

func (p *pr) ParseBranch() (Node, error) {
	if _, err := p.EatToken(lexer.BRANCH); err != nil {
		return nil, err
	}

	if _, err := p.EatToken(lexer.O_CUR_BRK); err != nil {
		return nil, err
	}

	item, err := p.ParseBranchItem()
	if err != nil {
		return nil, err
	}

	if _, err := p.EatToken(lexer.C_CUR_BRK); err != nil {
		return nil, err
	}

	return item, nil
}

func (p *pr) ParseBlockExpr() (Node, error) {
	if _, err := p.EatToken(lexer.O_CUR_BRK); err != nil {
		return nil, err
	}
	exprs, err := p.ParseExprs(lexer.TagCCurBrk)
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.C_CUR_BRK); err != nil {
		return nil, err
	}
	return BlockExpression{Body: exprs}, nil
}

func (p *pr) ParseDefer() (Node, error) {
	if _, err := p.EatToken(lexer.DEFER); err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.O_CUR_BRK); err != nil {
		return nil, err
	}
	exprs, err := p.ParseExprs(lexer.TagCCurBrk)
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.C_CUR_BRK); err != nil {
		return nil, err
	}
	block := BlockExpression{Body: exprs}
	return DeferExpression{Block: block}, nil
}

func (p *pr) ParseIf() (Node, error) {
	if _, err := p.EatToken(lexer.IF); err != nil {
		return nil, err
	}
	test, err := p.ParseBoolExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.O_CUR_BRK); err != nil {
		return nil, err
	}
	body, err := p.ParseExprs(lexer.TagCCurBrk)
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.C_CUR_BRK); err != nil {
		return nil, err
	}
	if p.GetLookahead().GetTag().Id == lexer.ELSE {
		euze, err := p.ParseElse()
		return IfExpression{test, body, euze}, err
	}
	return IfExpression{test, body, nil}, nil
}

func (p *pr) ParseElse() (*ElseExpression, error) {
	if _, err := p.EatToken(lexer.ELSE); err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.O_CUR_BRK); err != nil {
		return nil, err
	}
	body, err := p.ParseExprs(lexer.TagCCurBrk)
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.C_CUR_BRK); err != nil {
		return nil, err
	}
	return &ElseExpression{body}, nil
}

func (p *pr) ParseIdent() (Node, error) {
	if _, err := p.EatToken(lexer.IDENT); err != nil {
		return nil, err
	}
	id, err := p.EatToken(lexer.ID)
	if len(id.GetMatch()) == 0 {
		return nil, lexer.NewError(id, "missing identifier name at line: %d, column %d", id.GetLine(), id.GetColumn())
	}
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.ASSIGN); err != nil {
		return nil, err
	}
	expr, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	return IdentLiteral{string(id.GetMatch()), id, expr}, nil
}

func (p *pr) ParseExpr() (Node, error) {
	lookahead := p.GetLookahead()
	if lookahead == nil {
		return nil, fmt.Errorf("unexpected end of input")
	}
	if format, ok := printFormats[lookahead.GetTag().Id]; ok {
		return p.ParsePrint(lookahead.GetTag().Id, format)
	}
	if lookahead.GetTag().Id == lexer.ASSERT {
		return p.ParseAssert()
	}
	if lookahead.GetTag().Id == lexer.O_CUR_BRK {
		return p.ParseBlockExpr()
	}
	if lookahead.GetTag().Id == lexer.IF {
		return p.ParseIf()
	}
	if lookahead.GetTag().Id == lexer.BRANCH {
		return p.ParseBranch()
	}
	if lookahead.GetTag().Id == lexer.DEFER {
		return p.ParseDefer()
	}
	if lookahead.GetTag().Id == lexer.IDENT {
		return p.ParseIdent()
	}
	if lookahead.GetTag().Id == lexer.PULL {
		return p.ParsePull()
	}
	if lookahead.GetTag().Id == lexer.HEAD {
		return p.ParseHead()
	}
	if lookahead.GetTag().Id == lexer.TAIL {
		return p.ParseTail()
	}
	if lookahead.GetTag().Id == lexer.PUSH {
		return p.ParsePush()
	}
	return p.ParseBoolExpr()
}

// printFormats maps each print builtin to the reading it asks for.
var printFormats = map[string]PrintFormat{
	lexer.PRINTB: PrintBytes,
	lexer.PRINTC: PrintChars,
	lexer.PRINTD: PrintDecimal,
}

func (p *pr) ParsePrint(token string, format PrintFormat) (Node, error) {
	if _, err := p.EatToken(token); err != nil {
		return nil, err
	}
	expr, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	return PrintStatement{Format: format, Param: expr}, nil
}

func (p *pr) ParseAssert() (Node, error) {
	// Validate that assert can only be used in .test.ar files
	if !strings.HasSuffix(p.filename, ".test.ar") {
		lookahead := p.GetLookahead()
		return nil, lexer.NewError(lookahead, "assert can only be used in .test.ar files (at line %d, column %d)", lookahead.GetLine(), lookahead.GetColumn())
	}

	t, err := p.EatToken(lexer.ASSERT)
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.O_PAREN); err != nil {
		return nil, err
	}
	condition, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.COMMA); err != nil {
		return nil, err
	}
	message, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.EatToken(lexer.C_PAREN); err != nil {
		return nil, err
	}

	return AssertStatement{
		Condition: condition,
		Message:   message,
		Token:     t,
	}, nil
}

// ParseFeed parses the builtin "feed": feed(index), or feed index without parentheses.
func (p *pr) ParseFeed() (Node, error) {
	if _, err := p.EatToken(lexer.FEED); err != nil {
		return nil, err
	}
	if p.GetLookahead().GetTag().Id == lexer.O_PAREN {
		if _, err := p.EatToken(lexer.O_PAREN); err != nil {
			return nil, err
		}
		nth, err := p.ParseNumber()
		if err != nil {
			return nil, err
		}
		if _, err := p.EatToken(lexer.C_PAREN); err != nil {
			return nil, err
		}
		return FeedExpression{nth}, nil
	}
	nth, err := p.ParseNumber()
	if err != nil {
		return nil, err
	}
	return FeedExpression{nth}, nil
}

func (p *pr) ParseExprs(t lexer.Tag) ([]Node, error) {
	exprs := make([]Node, 0)
	for {
		lookahead := p.GetLookahead()
		if lookahead == nil || lookahead.GetTag().Id == t.Id {
			break
		}
		expr, err := p.ParseExpr()
		if err != nil {
			return exprs, err
		}
		if _, err := p.EatToken(lexer.SEMICOLON); err != nil {
			return exprs, err
		}
		exprs = append(exprs, expr)
	}
	return exprs, nil
}

func (p *pr) GetLookahead() lexer.Token {
	if p.cursor >= len(p.tokens) {
		return nil
	}
	return p.tokens[p.cursor]
}

func (p *pr) EatToken(tokenId string) (lexer.Token, error) {
	currtok := p.GetLookahead()

	if currtok == nil {
		return nil, nil
	}

	if tokenId != currtok.GetTag().Id {
		return currtok, lexer.NewError(currtok, "unexpected token %s at line %d and column %d", currtok.GetMatch(), currtok.GetLine(), currtok.GetColumn())
	}

	p.cursor++

	return currtok, nil
}

func (p *pr) Parse() (AST, error) {
	nodes, err := p.ParseExprs(lexer.TagEOF)
	if err != nil {
		return AST{}, err
	}

	ast := AST{Filename: p.filename, Nodes: nodes}

	if p.logger != nil {
		if _, err := p.logger.JSON(ast); err != nil {
			return AST{}, err
		}
	}

	return ast, nil
}

type NewParserOptions struct {
	// Filename is the source path. It decides file-scoped rules: assert is only
	// accepted inside *.test.ar.
	Filename      string
	Tokens        []lexer.Token
	EnableLogging bool
	// TapeSize is the width in bytes of every value. Zero means the default (8).
	TapeSize int
}

func New(opts NewParserOptions) Parser {
	return &pr{
		filename: opts.Filename,
		cursor:   0,
		tokens:   opts.Tokens,
		logger:   NewLogger(opts.EnableLogging),
		tapeSize: byteutil.TapeSize(opts.TapeSize),
	}
}
