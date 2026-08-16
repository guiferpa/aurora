package textdoc

import (
	"errors"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/lsp"
	"github.com/guiferpa/aurora/parser"
)

// Severity levels for diagnostics.
const (
	SeverityError       = 1
	SeverityWarning     = 2
	SeverityInformation = 3
	SeverityHint        = 4
)

// Analysis is one pass of the compiler front end over a single document: the token
// stream, the parsed namespace (nil when parsing failed) and a position mapper.
//
// The server analyses the open document alone, which is also the compiler's unit: there is
// no module system yet (see docs/module_system_design.md).
type Analysis struct {
	Source string
	Mapper *lsp.Mapper
	Tokens []lexer.Token
	AST    *parser.AST
	Err    error
}

// Analyze lexes and parses source. filename decides file-scoped rules — the parser only
// accepts "assert" inside *.test.ar — so callers should pass the path behind the URI.
func Analyze(filename, source string) *Analysis {
	analysis := &Analysis{Source: source, Mapper: lsp.NewMapper(source)}

	tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens([]byte(source))
	analysis.Tokens = tokens
	if err != nil {
		analysis.Err = err
		return analysis
	}

	ast, err := parser.New(parser.NewParserOptions{
		Filename: filename,
		Tokens:   tokens,
	}).Parse()
	if err != nil {
		analysis.Err = err
		return analysis
	}
	analysis.AST = &ast

	return analysis
}

// Diagnostics reports the failure of this pass, if any.
//
// The parser stops at the first error, so at most one diagnostic comes out of a pass;
// fixing it reveals the next one.
func (a *Analysis) Diagnostics() Diagnostics {
	diagnostics := Diagnostics{}
	if a.Err == nil {
		return diagnostics
	}

	source := "aurora"
	rng := lsp.Range{}

	// Position comes from the structured error carried by the lexer and the parser,
	// so the underline covers the offending token instead of guessing from the message.
	var perr *lexer.Error
	if errors.As(a.Err, &perr) {
		rng = a.rangeFor(perr.Offset, perr.Length)
	}

	return append(diagnostics, Diagnostic{
		Range:    rng,
		Severity: SeverityError,
		Source:   source,
		Message:  a.Err.Error(),
	})
}

// rangeFor converts a byte span into a range, backing up to the last meaningful character
// when the span would be empty. Errors reported against the EOF token — a missing
// semicolon, an unclosed block — otherwise produce a zero-width marker past the end of the
// document, which clients draw as nothing at all.
func (a *Analysis) rangeFor(offset, length int) lsp.Range {
	if length > 0 && offset >= 0 && offset+length <= len(a.Source) {
		return a.Mapper.Range(offset, length)
	}

	end := len(a.Source)
	if offset >= 0 && offset < end {
		end = offset
	}
	for end > 0 && isSpace(a.Source[end-1]) {
		end--
	}
	if end == 0 {
		return a.Mapper.Range(0, 0)
	}
	return a.Mapper.Range(end-1, 1)
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// TokenAt returns the token under an LSP position, or nil when the position sits on
// whitespace or past the end of the document.
func (a *Analysis) TokenAt(pos lsp.Position) lexer.Token {
	offset := a.Mapper.Offset(pos)
	for _, token := range a.Tokens {
		start := token.GetCursor()
		end := start + len(token.GetMatch())
		if offset >= start && offset < end {
			return token
		}
	}
	return nil
}

// ValidateCode is the entry point used by the didOpen and didChange handlers.
func ValidateCode(filename, source string) Diagnostics {
	return Analyze(filename, source).Diagnostics()
}

// HoverInfo describes what sits under the cursor: the description the lexer already
// carries for keywords, or the declaration of an identifier when it can be found.
func HoverInfo(filename, source string, pos lsp.Position) string {
	analysis := Analyze(filename, source)
	token := analysis.TokenAt(pos)
	if token == nil {
		return ""
	}

	tag := token.GetTag()
	match := string(token.GetMatch())

	if tag.Description != "" {
		return tag.Description
	}

	switch tag.Id {
	case lexer.NUMBER:
		return "number: " + match
	case lexer.STRING:
		return "text: " + match + "\nits bytes, in one tape"
	case lexer.TRUE, lexer.FALSE:
		return "boolean: " + match
	case lexer.ID:
		// A struct name or a field read out of one: the directive is what says these are
		// anything other than a name, so it is what hover has to answer with.
		if shape, fields, index := scanStructs(analysis.Tokens).structAt(analysis.Tokens, token); shape != "" {
			if index < 0 {
				return "struct " + shape + "\nfields: " + strings.Join(fields, ", ")
			}
			return "field " + match + " of struct " + shape + "\nreads tape " + strconv.Itoa(index) + " of the run"
		}
		if analysis.AST == nil {
			return "identifier: " + match
		}
		if def := FindIdent(analysis.AST.Nodes, match); def != nil {
			return "identifier: " + match + "\nvalue: " + describeNode(def.Value)
		}
		return "identifier: " + match
	}

	return ""
}

// CompletionItemsFor offers the language keywords plus the identifiers declared in the
// document being edited — or, right after a dot on a value with a known shape, the fields
// of that struct and nothing else.
//
// It takes a position for that last case: what to offer depends on what sits in front of
// the cursor, and a document being edited usually does not parse.
func CompletionItemsFor(filename, source string, pos lsp.Position, snippets bool) []CompletionItem {
	analysis := Analyze(filename, source)
	shapes := scanStructs(analysis.Tokens)

	// Right after a dot the fields are the answer, and the only one: nothing else can
	// follow it.
	if fields := shapes.fieldsBefore(analysis.Tokens, analysis.Mapper.Offset(pos)); len(fields) > 0 {
		return fieldCompletions(fields)
	}

	items := make([]CompletionItem, 0)
	for _, tag := range lexer.GetProcessableTags() {
		items = append(items, keywordCompletion(tag, snippets))
	}
	items = append(items, structCompletions(shapes, snippets)...)

	if analysis.AST == nil {
		return items
	}
	for _, ident := range CollectIdents(analysis.AST.Nodes) {
		items = append(items, CompletionItem{
			Label:  ident.Id,
			Detail: describeNode(ident.Value),
			Kind:   Variable,
		})
	}
	return items
}

// FindIdent looks for the declaration of name, walking into the expression forms that
// carry a body.
func FindIdent(nodes []parser.Node, name string) *parser.IdentLiteral {
	for _, node := range nodes {
		if ident, ok := node.(parser.IdentLiteral); ok {
			if ident.Id == name {
				return &ident
			}
			if found := FindIdent([]parser.Node{ident.Value}, name); found != nil {
				return found
			}
		}
		if found := FindIdent(childrenOf(node), name); found != nil {
			return found
		}
	}
	return nil
}

// CollectIdents lists every identifier declared in the document, outermost first.
func CollectIdents(nodes []parser.Node) []parser.IdentLiteral {
	idents := make([]parser.IdentLiteral, 0)
	for _, node := range nodes {
		if ident, ok := node.(parser.IdentLiteral); ok {
			idents = append(idents, ident)
			idents = append(idents, CollectIdents([]parser.Node{ident.Value})...)
			continue
		}
		idents = append(idents, CollectIdents(childrenOf(node))...)
	}
	return idents
}

func childrenOf(node parser.Node) []parser.Node {
	switch n := node.(type) {
	case parser.BlockExpression:
		return n.Body
	case parser.DeferExpression:
		return n.Block.Body
	case parser.IfExpression:
		body := make([]parser.Node, 0, len(n.Body)+1)
		body = append(body, n.Body...)
		if n.Else != nil {
			body = append(body, n.Else.Body...)
		}
		return body
	default:
		return nil
	}
}

func describeNode(node parser.Node) string {
	switch node.(type) {
	case parser.NumberLiteral:
		return "number"
	case parser.BooleanLiteral:
		return "boolean"
	case parser.TextLiteral:
		return "text"
	case parser.IdentifierLiteral:
		return "identifier"
	case parser.TapeBracketExpression:
		return "tape"
	case parser.PullExpression, parser.PushExpression, parser.HeadExpression, parser.TailExpression:
		return "tape operation"
	case parser.BinaryExpression:
		return "arithmetic expression"
	case parser.RelativeExpression, parser.BooleanExpression:
		return "boolean expression"
	case parser.IfExpression:
		return "if expression"
	case parser.BlockExpression:
		return "scope"
	case parser.DeferExpression:
		return "deferred scope"
	case parser.CalleeLiteral:
		return "call"
	default:
		return "expression"
	}
}

// PathFromURI turns a file:// URI into a filesystem path. The path decides whether the
// parser accepts "assert", so it has to survive percent-encoding.
func PathFromURI(uri lsp.URI) string {
	raw := string(uri)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "file" {
		return raw
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		path = parsed.Path
	}
	// Windows: file:///c:/dir/main.ar parses with a leading slash.
	if strings.HasPrefix(path, "/") && filepath.VolumeName(strings.TrimPrefix(path, "/")) != "" {
		path = strings.TrimPrefix(path, "/")
	}
	return filepath.FromSlash(path)
}
