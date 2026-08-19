package textdoc

import (
	"errors"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/loader"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
	"github.com/guiferpa/aurora/wire/token"
)

// Severity levels for diagnostics.
const (
	SeverityError       = 1
	SeverityWarning     = 2
	SeverityInformation = 3
	SeverityHint        = 4
)

// Analysis is one pass of the compiler front end over a single document: the tk
// stream, the parsed tree (nil when parsing failed), a position mapper, and the modules the
// document imports.
//
// The modules are read only when the document parses, because until it does there is no list
// of imports to read. They are the one thing here that comes from outside the document, and
// they arrive through a port: see Resolve.
type Analysis struct {
	Source  string
	Mapper  *lsp.Mapper
	Tokens  []token.Token
	AST     *ast.AST
	Err     error
	Modules []module.Module
	// ModuleErr is what the imports had to say: a module that is not there, a circle, or a
	// name a module does not have. It is kept apart from Err because it is found after the
	// parse and only when the parse worked.
	ModuleErr error
}

// module answers the module of a specifier among the ones this document imports.
func (a *Analysis) module(specifier string) (module.Module, bool) {
	for _, each := range a.Modules {
		if string(each.ID) == specifier {
			return each, true
		}
	}
	return module.Module{}, false
}

// Document is what an analysis reads: the source, the path behind the URI — which decides
// file-scoped rules, since the parser only accepts "assert" inside *.test.ar — and how wide
// a value is.
//
// The width is given rather than discovered because finding it is the host's business and
// each host finds it differently: the language server walks up from the file to the project
// manifest, and the playground reads the control on the page. This package takes values and
// answers with values, which is also what lets it run where there is no filesystem at all.
//
// A zero width means the language's default.
type Document struct {
	Filename string
	Source   string
	TapeSize int
}

// Analyze lexes and parses a document.
func (s *Session) Analyze(doc Document) *Analysis {
	analysis := &Analysis{Source: doc.Source, Mapper: lsp.NewMapper(doc.Source)}

	tokens, err := s.lexer.GetFilledTokens([]byte(doc.Source))
	analysis.Tokens = tokens
	if err != nil {
		analysis.Err = err
		return analysis
	}

	// What the document imports is read from the tokens, before anything is parsed: a
	// document being edited is broken most of the time, and what is inside a module is
	// wanted exactly then. What that costs is one read of each imported file.
	if s.resolve != nil {
		modules, err := s.resolve(doc, scanUses(tokens))
		analysis.Modules, analysis.ModuleErr = modules, err
	}

	// How wide a value is decides what fits in one, so the document is read in the dialect it
	// belongs to rather than in the default.
	tree, err := s.parser.Parse(parser.ParseInput{
		Filename: doc.Filename,
		Tokens:   tokens,
		TapeSize: doc.TapeSize,
	})
	if err != nil {
		analysis.Err = err
		return analysis
	}
	analysis.AST = &tree

	// The names have to be there too, and only a parsed document has names to check.
	if analysis.ModuleErr == nil && s.resolve != nil {
		analysis.ModuleErr = loader.Check(append(analysis.Modules, module.Module{ID: "", Tree: tree}))
	}

	return analysis
}

// Diagnostics reports the failure of this pass, if any.
//
// The parser stops at the first error, so at most one diagnostic comes out of a pass;
// fixing it reveals the next one.
func (a *Analysis) Diagnostics() Diagnostics {
	diagnostics := Diagnostics{}

	// The parse comes first: a document that does not parse has no imports to be wrong
	// about, and saying both at once would be saying the second about a tree that is not
	// there.
	failure := a.Err
	if failure == nil {
		failure = a.ModuleErr
	}
	if failure == nil {
		return diagnostics
	}

	source := "aurora"
	rng := lsp.Range{}

	// Position comes from the structured error carried by the lexer and the parser,
	// so the underline covers the offending tk instead of guessing from the message.
	var perr *token.Error
	if errors.As(failure, &perr) {
		rng = a.rangeFor(perr.Offset, perr.Length)
	}

	return append(diagnostics, Diagnostic{
		Range:    rng,
		Severity: SeverityError,
		Source:   source,
		Message:  failure.Error(),
	})
}

// rangeFor converts a byte span into a range, backing up to the last meaningful character
// when the span would be empty. Errors reported against the EOF tk — a missing
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

// TokenAt returns the tk under an LSP position, or nil when the position sits on
// whitespace or past the end of the document.
func (a *Analysis) TokenAt(pos lsp.Position) token.Token {
	offset := a.Mapper.Offset(pos)
	for _, tk := range a.Tokens {
		start := tk.GetCursor()
		end := start + len(tk.GetMatch())
		if offset >= start && offset < end {
			return tk
		}
	}
	return nil
}

// ValidateCode is the entry point used by the didOpen and didChange handlers.
func (s *Session) ValidateCode(doc Document) Diagnostics {
	return s.Analyze(doc).Diagnostics()
}

// HoverInfo describes what sits under the cursor: the description the lexer already
// carries for keywords, or the declaration of an identifier when it can be found.
func (s *Session) HoverInfo(doc Document, pos lsp.Position) string {
	analysis := s.Analyze(doc)
	tk := analysis.TokenAt(pos)
	if tk == nil {
		return ""
	}

	tag := tk.GetTag()
	match := string(tk.GetMatch())

	if tag.Description != "" {
		return tag.Description
	}

	switch tag.Id {
	case token.NUMBER:
		return "number: " + match
	case token.STRING:
		return "text: " + match + "\nits bytes, in one tape"
	case token.TRUE, token.FALSE:
		return "boolean: " + match
	case token.ID:
		// A module is not a value, so it is answered for before anything looks for one.
		if info := aliasesOf(scanUses(analysis.Tokens)).describe(analysis.Tokens, tk); info != "" {
			return info + describeExport(analysis, tk)
		}
		// A struct name or a field read out of one: the declaration is what says these are
		// anything other than a name, so it is what hover has to answer with.
		if shape, fields, index := scanStructs(analysis.Tokens).structAt(analysis.Tokens, tk); shape != "" {
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
func (s *Session) CompletionItemsFor(doc Document, pos lsp.Position, snippets bool) []CompletionItem {
	analysis := s.Analyze(doc)
	shapes := scanStructs(analysis.Tokens)

	// Right after a dot the fields are the answer, and the only one: nothing else can
	// follow it.
	offset := analysis.Mapper.Offset(pos)
	if fields := shapes.fieldsBefore(analysis.Tokens, offset); len(fields) > 0 {
		return fieldCompletions(fields)
	}

	// After a dot on a module alias, what can follow is what that module declared, and
	// nothing else — the same rule as a struct's fields, answered from another file.
	aliases := aliasesOf(scanUses(analysis.Tokens))
	if specifier, isModule := aliases.moduleBefore(analysis.Tokens, offset); isModule {
		return exportCompletions(analysis, specifier)
	}

	items := make([]CompletionItem, 0)
	for _, tag := range token.GetProcessableTags() {
		items = append(items, keywordCompletion(tag, snippets))
	}
	items = append(items, structCompletions(shapes, snippets)...)
	items = append(items, moduleCompletions(aliases)...)

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
func FindIdent(nodes []ast.Node, name string) *ast.IdentLiteral {
	for _, node := range nodes {
		if ident, ok := node.(ast.IdentLiteral); ok {
			if ident.Id == name {
				return &ident
			}
			if found := FindIdent([]ast.Node{ident.Value}, name); found != nil {
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
func CollectIdents(nodes []ast.Node) []ast.IdentLiteral {
	idents := make([]ast.IdentLiteral, 0)
	for _, node := range nodes {
		if ident, ok := node.(ast.IdentLiteral); ok {
			idents = append(idents, ident)
			idents = append(idents, CollectIdents([]ast.Node{ident.Value})...)
			continue
		}
		idents = append(idents, CollectIdents(childrenOf(node))...)
	}
	return idents
}

func childrenOf(node ast.Node) []ast.Node {
	switch n := node.(type) {
	case ast.BlockExpression:
		return n.Body
	case ast.DeferExpression:
		return n.Block.Body
	case ast.IfExpression:
		body := make([]ast.Node, 0, len(n.Body)+1)
		body = append(body, n.Body...)
		if n.Else != nil {
			body = append(body, n.Else.Body...)
		}
		return body
	default:
		return nil
	}
}

func describeNode(node ast.Node) string {
	switch node.(type) {
	case ast.NumberLiteral:
		return "number"
	case ast.BooleanLiteral:
		return "boolean"
	case ast.TextLiteral:
		return "text"
	case ast.IdentifierLiteral:
		return "identifier"
	case ast.TapeBracketExpression:
		return "tape"
	case ast.PullExpression, ast.PushExpression, ast.HeadExpression, ast.TailExpression:
		return "tape operation"
	case ast.BinaryExpression:
		return "arithmetic expression"
	case ast.RelativeExpression, ast.BooleanExpression:
		return "boolean expression"
	case ast.IfExpression:
		return "if expression"
	case ast.BlockExpression:
		return "scope"
	case ast.DeferExpression:
		return "deferred scope"
	case ast.CalleeLiteral:
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
