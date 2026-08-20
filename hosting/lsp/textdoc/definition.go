package textdoc

import (
	"encoding/json"

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
func (s *Session) DefinitionFor(doc Document, pos lsp.Position) (Definition, bool) {
	analysis := s.Analyze(doc)
	subject := analysis.TokenAt(pos)
	if subject == nil || subject.GetTag().Id != token.ID {
		return Definition{}, false
	}

	found := declaredIn(analysis.Tokens, subject, shapesOf(analysis, aliasesOf(parser.ScanUses(analysis.Tokens))))
	if found == nil {
		return Definition{}, false
	}
	return Definition{
		Filename: doc.Filename,
		Range:    analysis.Mapper.Range(found.GetCursor(), len(found.GetMatch())),
	}, true
}

// declaredIn answers the token that declares the subject, among the tokens of the file it was
// written in.
//
// A field is looked for first because the same name may be both — `p.x` where an `ident x`
// exists elsewhere is the field, and the dot is what says so.
func declaredIn(tokens []token.Token, subject token.Token, shapes shapeTable) token.Token {
	if owner := ownerOf(tokens, subject); owner != nil {
		return fieldToken(tokens, shapes.reads[string(owner.GetMatch())], string(subject.GetMatch()))
	}
	return bindingToken(tokens, subject)
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

// bindingToken answers where a plain name was declared: `shape Point`, or the nearest
// `ident name` written before it.
//
// Nearest before, because that is the one the language would find: a name is bound before it
// is used, and a second binding of the same name shadows the first from where it is written.
// A binding that only appears later is answered with anyway — a deferred scope runs when it
// is called, so a body can name something written under it, and a jump to the wrong end of a
// file is a better answer than none.
func bindingToken(tokens []token.Token, subject token.Token) token.Token {
	name := string(subject.GetMatch())

	var before, after token.Token
	for i := 1; i < len(tokens); i++ {
		if string(tokens[i].GetMatch()) != name || tokens[i].GetTag().Id != token.ID {
			continue
		}
		if declares := tokens[i-1].GetTag().Id; declares != token.IDENT && declares != token.SHAPE {
			continue
		}
		if tokens[i].GetCursor() <= subject.GetCursor() {
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
