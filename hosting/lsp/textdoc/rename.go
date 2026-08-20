package textdoc

import (
	"encoding/json"
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/wire/token"
)

// Renaming a name everywhere it is written.
//
// It is the one thing here that writes rather than answers, which is why it refuses more than
// it accepts. An editor that offers a rename it cannot finish is worse than one that offers
// none: the half that was renamed compiles, the half that was not is somebody else's file,
// and nothing on screen says which is which.

// ErrNotRenameable is what a refusal is, so a host can tell it from a failure. The message
// says which name and why, and it is written for whoever is holding the cursor.
var ErrNotRenameable = errors.New("not renameable")

type RenameParams struct {
	PositionParams
	NewName string `json:"newName"`
}

type RenameRequest struct {
	lsp.Request
	Params RenameParams `json:"params"`
}

type RenameResponse struct {
	lsp.Response
	Result WorkspaceEdit `json:"result"`
}

// PrepareRenameRequest is what a client asks before it opens the box: which range is about to
// be renamed, and whether it can be at all.
type PrepareRenameRequest struct {
	lsp.Request
	Params PositionParams `json:"params"`
}

type PrepareRenameResponse struct {
	lsp.Response
	Result lsp.Range `json:"result"`
}

func ParseRenameRequest(contents []byte) (*RenameRequest, error) {
	var req RenameRequest
	if err := json.Unmarshal(contents, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func ParsePrepareRenameRequest(contents []byte) (*PrepareRenameRequest, error) {
	var req PrepareRenameRequest
	if err := json.Unmarshal(contents, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func NewRenameResponse(id int, edit WorkspaceEdit) RenameResponse {
	return RenameResponse{Response: lsp.Response{RPC: "2.0", ID: &id}, Result: edit}
}

func NewPrepareRenameResponse(id int, at lsp.Range) PrepareRenameResponse {
	return PrepareRenameResponse{Response: lsp.Response{RPC: "2.0", ID: &id}, Result: at}
}

// A Rename is every place one name is written, in the file that writes it.
type Rename struct {
	Filename string
	// Ranges holds the declaration and every use of it, in the order they are written.
	Ranges []lsp.Range
}

// PrepareRename answers the range of the name under the cursor, and the reason when it is a
// name that cannot be renamed. It is asked before the editor opens its box, so the refusal
// arrives before somebody types a new name rather than after.
func (s *Session) PrepareRename(doc Document, pos lsp.Position) (lsp.Range, error) {
	analysis := s.Analyze(doc)
	subject, _, err := s.renameable(analysis, pos)
	if err != nil {
		return lsp.Range{}, err
	}
	return rangeOf(analysis.Mapper, subject), nil
}

// RenameFor answers every place the name under the cursor is written, so the editor can
// change all of them at once, and the reason when it must not.
func (s *Session) RenameFor(doc Document, pos lsp.Position, newName string) (Rename, error) {
	analysis := s.Analyze(doc)
	subject, table, err := s.renameable(analysis, pos)
	if err != nil {
		return Rename{}, err
	}
	declaration := declarationOf(table, subject)
	if err := s.nameable(analysis.Tokens, declaration, newName); err != nil {
		return Rename{}, err
	}

	ranges := make([]lsp.Range, 0)
	for _, tk := range analysis.Tokens {
		if tk.GetCursor() != declaration.GetCursor() && !means(table, tk, declaration) {
			continue
		}
		ranges = append(ranges, rangeOf(analysis.Mapper, tk))
	}
	return Rename{Filename: doc.Filename, Ranges: ranges}, nil
}

// renameable answers the name under the cursor and what the file's names mean, or the reason
// this one is not something to rename.
//
// Both requests ask it, because a client that asks nothing first has to get the same answer
// from the rename itself.
func (s *Session) renameable(analysis *Analysis, pos lsp.Position) (token.Token, scopeTable, error) {
	subject := analysis.TokenAt(pos)
	if subject == nil || subject.GetTag().Id != token.ID {
		return nil, scopeTable{}, fmt.Errorf("%w: there is no name here", ErrNotRenameable)
	}

	table := scopesOf(analysis.Tokens)
	declaration := declarationOf(table, subject)
	if declaration == nil {
		return nil, scopeTable{}, fmt.Errorf("%w: %s is not declared in this file", ErrNotRenameable, subject.GetMatch())
	}
	// A name at the top of a file is what a module offers, and what a module offers another
	// file may be reaching for by now. Renaming it here would be half of the change, and the
	// half that is missing is in a file nobody has looked at.
	if table.tops[declaration.GetCursor()] {
		return nil, scopeTable{}, fmt.Errorf("%w: %s is bound at the top of this file, so another file may be importing it", ErrNotRenameable, subject.GetMatch())
	}
	return subject, table, nil
}

// nameable answers whether a new name is one the language would accept.
//
// Whether it is a name at all is asked of the lexer rather than of a rule written twice: a
// keyword, a number, a space or a dot all come back as something other than one name. The
// capital is asked here, because a shape's name has to start with one and the editor refusing
// it now is better than the parser refusing it after every occurrence has been rewritten.
func (s *Session) nameable(tokens []token.Token, declaration token.Token, newName string) error {
	read, err := s.lexer.GetFilledTokens([]byte(newName))
	if err != nil || len(read) != 2 || read[0].GetTag().Id != token.ID || read[1].GetTag().Id != token.EOF {
		return fmt.Errorf("%w: %q is not a name", ErrNotRenameable, newName)
	}
	if i := indexOf(tokens, declaration); i > 0 && tokens[i-1].GetTag().Id == token.SHAPE && !capitalized(newName) {
		return fmt.Errorf("%w: %s names a shape, and a shape's name starts with a capital letter", ErrNotRenameable, newName)
	}
	return nil
}

// capitalized says whether a name is written the way a shape's name has to be, which is the
// parser's rule read here so the refusal arrives before the rewrite rather than after it.
func capitalized(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(first)
}

// declarationOf answers the declaration a name means, which is itself when the cursor is on
// the declaration.
func declarationOf(table scopeTable, subject token.Token) token.Token {
	if _, declared := table.tops[subject.GetCursor()]; declared {
		return subject
	}
	return table.means[subject.GetCursor()]
}

// means answers whether a token is a use of this declaration.
func means(table scopeTable, tk, declaration token.Token) bool {
	found, resolved := table.means[tk.GetCursor()]
	return resolved && found.GetCursor() == declaration.GetCursor()
}
