package token

import (
	"strings"
	"testing"
)

// The token chain is what the lexer hands the parser, what the tree holds on to, and what a
// host underlines when something is wrong. Until it moved here it had no tests of its own:
// the lexer tested what it read and the language server tested what it showed, and the thing
// the three agree on was covered only sideways.

// A token is built once and read many times, in places that never saw the source: the parser
// asks what it matched, the tree keeps it, the language server asks where it was.
func TestNewKeepsEveryPart(t *testing.T) {
	tok := New([]byte("ident"), TagIdent, 3, 7, 42)

	if got := string(tok.GetMatch()); got != "ident" {
		t.Errorf("match is %q, want ident", got)
	}
	if got := tok.GetTag(); got != TagIdent {
		t.Errorf("tag is %v, want %v", got, TagIdent)
	}
	if got := tok.GetLine(); got != 3 {
		t.Errorf("line is %d, want 3", got)
	}
	if got := tok.GetColumn(); got != 7 {
		t.Errorf("column is %d, want 7", got)
	}
	if got := tok.GetCursor(); got != 42 {
		t.Errorf("cursor is %d, want 42", got)
	}
}

// Every tag offered for completion has a word to offer. One without would reach the editor as
// an empty suggestion, which is worse than no suggestion at all.
func TestEveryProcessableTagHasAKeyword(t *testing.T) {
	tags := GetProcessableTags()

	if len(tags) == 0 {
		t.Fatal("no tag is offered at all")
	}
	for _, tag := range tags {
		if tag.Keyword == "" {
			t.Errorf("tag %s is offered with nothing to type", tag.Id)
		}
		if tag.Id == "" {
			t.Errorf("the tag for %q has no id", tag.Keyword)
		}
	}
}

// Two tags answering to the same word would make a completion list offer the same thing
// twice, and the second one would be unreachable.
func TestNoTwoProcessableTagsShareAKeyword(t *testing.T) {
	seen := make(map[string]string)

	for _, tag := range GetProcessableTags() {
		if first, taken := seen[tag.Keyword]; taken {
			t.Errorf("%q is the word for both %s and %s", tag.Keyword, first, tag.Id)
			continue
		}
		seen[tag.Keyword] = tag.Id
	}
}

// An error carries where it happened so a caller can point at it without reading the message.
func TestNewErrorTakesItsPlaceFromTheToken(t *testing.T) {
	err := NewError(New([]byte("printx"), TagId, 2, 5, 17), "unexpected token %s", "printx")

	if err.Message != "unexpected token printx" {
		t.Errorf("message is %q", err.Message)
	}
	if err.Line != 2 || err.Column != 5 {
		t.Errorf("placed at %d:%d, want 2:5", err.Line, err.Column)
	}
	if err.Offset != 17 {
		t.Errorf("offset is %d, want 17", err.Offset)
	}
	// The length is what the token matched, which is what an editor underlines.
	if err.Length != 6 {
		t.Errorf("length is %d, want the six bytes of printx", err.Length)
	}
	if !strings.Contains(err.Error(), "printx") {
		t.Errorf("Error() is %q", err.Error())
	}
}

// Not every failure has a token to blame — the end of a file has nothing to underline. It
// still has to answer, and with a length an editor can draw.
func TestNewErrorWithoutAToken(t *testing.T) {
	err := NewError(nil, "unexpected end of file")

	if err.Message != "unexpected end of file" {
		t.Errorf("message is %q", err.Message)
	}
	if err.Line != 0 || err.Column != 0 {
		t.Errorf("placed at %d:%d, want nowhere", err.Line, err.Column)
	}
	if err.Length != 1 {
		t.Errorf("length is %d, want one so there is something to draw", err.Length)
	}
}

// A token that matched nothing — the end of a file is one — still gets a length, for the
// same reason.
func TestNewErrorOnATokenThatMatchedNothing(t *testing.T) {
	err := NewError(New([]byte{}, TagEOF, 9, 1, 100), "unexpected end of file")

	if err.Length != 1 {
		t.Errorf("length is %d, want one", err.Length)
	}
	if err.Line != 9 {
		t.Errorf("line is %d, want 9", err.Line)
	}
}
