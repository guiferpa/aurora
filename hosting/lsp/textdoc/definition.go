package textdoc

import (
	"encoding/json"
	"strings"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/token"
)

// Where a name was declared, and how the editor is told.
//
// It is read from the tokens, like everything else here. A definition is asked for while
// somebody is reading rather than typing, so the document usually parses — but the tree is
// the wrong place to look anyway: what is wanted is where a name was *written*, and the tree
// keeps what a name means. The token stream keeps where.

type DefinitionParams struct {
	PositionParams
}

type DefinitionRequest struct {
	lsp.Request
	Params DefinitionParams `json:"params"`
}

// DefinitionResponse answers with one location, which is one of the three forms the protocol
// allows. A name is declared in one place in Aurora, so the list form would always hold one.
type DefinitionResponse struct {
	lsp.Response
	Result lsp.Location `json:"result"`
}

func ParseDefinitionRequest(contents []byte) (*DefinitionRequest, error) {
	var req DefinitionRequest
	if err := json.Unmarshal(contents, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func NewDefinitionResponse(id int, location lsp.Location) DefinitionResponse {
	return DefinitionResponse{
		Response: lsp.Response{RPC: "2.0", ID: &id},
		Result:   location,
	}
}

// A Definition is where a name was declared: which file, and the range of the name inside it.
//
// The file is a path rather than a URI because this package answers with values and never
// touches the world — which URI a path is depends on the host, and only the host knows.
type Definition struct {
	Filename string
	Range    lsp.Range
}

// DefinitionFor answers where the name under the cursor was declared, and false when nothing
// under it is a name or nothing declares it.
//
// Four things are names here, and each is declared somewhere a token can be found: a value
// bound with `ident`, a shape declared with `shape`, a field inside a shape's declaration,
// and the declaration itself — asking about a name where it was written answers with itself,
// which is what tells "declared here" from "not declared at all".
//
// The file it lands in is this one or a module's, and the question does not change between
// them: the other file is lexed and asked the same thing.
func (s *Session) DefinitionFor(doc Document, pos lsp.Position) (Definition, bool) {
	analysis := s.Analyze(doc)
	subject := analysis.TokenAt(pos)
	if subject == nil || subject.GetTag().Id != token.ID {
		return Definition{}, false
	}
	aliases := aliasesOf(parser.ScanUses(analysis.Tokens))

	// A use line is about a module wherever the cursor sits in it — on the keyword, on the
	// path or on the alias — and so is an alias anywhere else. The module is a file, so the
	// answer is its top.
	if alias, declared := aliasOfUse(analysis.Tokens, subject); declared {
		return s.inModule(analysis, aliases[alias], "")
	}
	if specifier, isModule := aliases[string(subject.GetMatch())]; isModule {
		return s.inModule(analysis, specifier, "")
	}
	// A name reached through an alias is declared in that module's file, and never here.
	if specifier, reached := moduleOfSubject(analysis, subject); reached {
		return s.inModule(analysis, specifier, string(subject.GetMatch()))
	}

	shapes := shapesOf(analysis, aliases)
	if owner := ownerOf(analysis.Tokens, subject); owner != nil {
		return s.fieldOf(analysis, aliases, doc.Filename, shapes.reads[string(owner.GetMatch())], string(subject.GetMatch()))
	}
	found := bindingOf(analysis.Tokens, string(subject.GetMatch()), subject.GetCursor())
	if found == nil {
		return Definition{}, false
	}
	return Definition{Filename: doc.Filename, Range: rangeOf(analysis.Mapper, found)}, true
}

// fieldOf answers where a field was named, in the file that declares the shape it belongs to.
// A shape reached through an alias carries it, and that is the only part the reader here has
// to undo: `g.Square` is Square, over there.
func (s *Session) fieldOf(analysis *Analysis, aliases moduleAliases, filename, shape, field string) (Definition, bool) {
	if alias, name, qualified := strings.Cut(shape, "."); qualified {
		specifier, isModule := aliases[alias]
		if !isModule {
			return Definition{}, false
		}
		tokens, mapper, filename, ok := s.readModule(analysis, specifier)
		if !ok {
			return Definition{}, false
		}
		return found(fieldToken(tokens, name, field), mapper, filename)
	}
	return found(fieldToken(analysis.Tokens, shape, field), analysis.Mapper, filename)
}

// inModule answers where a name is declared in another file, or the file itself when the name
// is empty — which is what an alias means: a module is a file, and there is nothing narrower
// to point at.
func (s *Session) inModule(analysis *Analysis, specifier, symbol string) (Definition, bool) {
	tokens, mapper, filename, ok := s.readModule(analysis, specifier)
	if !ok {
		return Definition{}, false
	}
	if symbol == "" {
		return Definition{Filename: filename, Range: lsp.LineRange(0, 0, 0)}, true
	}
	// From the top: what a module offers is bound at the top level of its file, and nothing
	// in the file that imports it says where.
	return found(bindingOf(tokens, symbol, 0), mapper, filename)
}

// readModule lexes a module's own file, so the same questions can be asked of it as of the
// document being edited. It answers nothing for a module that was never resolved — the one
// that is not there, and the whole file when there is no port to resolve through.
func (s *Session) readModule(analysis *Analysis, specifier string) ([]token.Token, *lsp.Mapper, string, bool) {
	module, imported := analysis.module(specifier)
	if !imported {
		return nil, nil, "", false
	}
	tokens, err := s.lexer.GetFilledTokens([]byte(module.Source))
	if err != nil {
		return nil, nil, "", false
	}
	return tokens, lsp.NewMapper(module.Source), module.Tree.Filename, true
}

// found turns a token into the answer, and a token nobody found into no answer.
func found(declaration token.Token, mapper *lsp.Mapper, filename string) (Definition, bool) {
	if declaration == nil {
		return Definition{}, false
	}
	return Definition{Filename: filename, Range: rangeOf(mapper, declaration)}, true
}

// rangeOf is the range a token covers in the file it was read from.
func rangeOf(mapper *lsp.Mapper, tk token.Token) lsp.Range {
	return mapper.Range(tk.GetCursor(), len(tk.GetMatch()))
}

// aliasOfUse answers the alias of the use declaration the subject sits inside, and false when
// it sits anywhere else.
//
// It reads the statement rather than the line: two declarations on one line are two
// statements, and the answer has to be about the one the cursor is in.
func aliasOfUse(tokens []token.Token, subject token.Token) (string, bool) {
	i := indexOf(tokens, subject)
	if i < 0 {
		return "", false
	}
	for ; i >= 0; i-- {
		if tokens[i].GetTag().Id == token.SEMICOLON {
			return "", false
		}
		if tokens[i].GetTag().Id == token.USE {
			break
		}
	}
	if i < 0 {
		return "", false
	}
	// `use a/b/c as x;`: the alias is the name after `as`, and it is the last thing the
	// declaration holds.
	for ; i < len(tokens)-1; i++ {
		if tokens[i].GetTag().Id == token.AS && tokens[i+1].GetTag().Id == token.ID {
			return string(tokens[i+1].GetMatch()), true
		}
		if tokens[i].GetTag().Id == token.SEMICOLON {
			break
		}
	}
	return "", false
}

// indexOf answers where a token sits in the stream it came from. Tokens are compared by where
// they start: the concrete type holds slices and cannot be compared with ==.
func indexOf(tokens []token.Token, subject token.Token) int {
	for i, tk := range tokens {
		if tk.GetCursor() == subject.GetCursor() {
			return i
		}
	}
	return -1
}

// ownerOf answers the name a token is read out of — the `p` of `p.x` — and nil when the token
// does not follow a dot.
//
// It is the same question ownerBefore asks of a cursor, from the other end: that one is for a
// dot somebody just typed, this one for a name already there.
func ownerOf(tokens []token.Token, subject token.Token) token.Token {
	for i, tk := range tokens {
		// Tokens are compared by where they start: the concrete type holds slices and cannot
		// be compared with ==.
		if tk.GetCursor() != subject.GetCursor() || i < 2 || tokens[i-1].GetTag().Id != token.DOT {
			continue
		}
		if owner := tokens[i-2]; owner.GetTag().Id == token.ID {
			return owner
		}
	}
	return nil
}

// bindingOf answers where a plain name was declared: `shape Point`, or the nearest
// `ident name` written before the cursor it is asked about.
//
// Nearest before, because that is the one the language would find: a name is bound before it
// is used, and a second binding of the same name shadows the first from where it is written.
// A binding that only appears later is answered with anyway — a deferred scope runs when it
// is called, so a body can name something written under it, and a jump to the wrong end of a
// file is a better answer than none.
func bindingOf(tokens []token.Token, name string, at int) token.Token {
	var before, after token.Token
	for i := 1; i < len(tokens); i++ {
		if string(tokens[i].GetMatch()) != name || tokens[i].GetTag().Id != token.ID {
			continue
		}
		if declares := tokens[i-1].GetTag().Id; declares != token.IDENT && declares != token.SHAPE {
			continue
		}
		if tokens[i].GetCursor() <= at {
			before = tokens[i]
			continue
		}
		if after == nil {
			after = tokens[i]
		}
	}
	if before != nil {
		return before
	}
	return after
}

// fieldToken answers where a field was named, which is inside the declaration of the shape it
// belongs to: the `y` of `shape Point { x, y }`.
func fieldToken(tokens []token.Token, shape, field string) token.Token {
	for i := 1; i < len(tokens); i++ {
		if tokens[i-1].GetTag().Id != token.SHAPE || string(tokens[i].GetMatch()) != shape {
			continue
		}
		for j := i + 1; j < len(tokens); j++ {
			if tokens[j].GetTag().Id == token.C_CUR_BRK {
				return nil
			}
			if tokens[j].GetTag().Id == token.ID && string(tokens[j].GetMatch()) == field {
				return tokens[j]
			}
		}
	}
	return nil
}
