package parser

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/wire/token"
)

// One parser answers for every source there is. It used to take its tokens at build time, so
// an instance parsed exactly one file — which is why no host could be handed a parser, only a
// way of making one. What follows is the property that changed.

// tokensOf lexes a source, which is what a parse is given.
func tokensOf(t *testing.T, source string) []token.Token {
	t.Helper()
	tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	return tokens
}

func TestOneParserReadsOneSourceAfterAnother(t *testing.T) {
	p := New(NewParserOptions{})

	first, err := p.Parse(ParseInput{Filename: "first.ar", Tokens: tokensOf(t, "1 + 2;")})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := p.Parse(ParseInput{Filename: "second.ar", Tokens: tokensOf(t, "3 * 4;\n5;")})
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	// Each tree is its own source's: a cursor left where the first parse stopped would read
	// the second from the middle, or from its end.
	if first.Filename != "first.ar" || second.Filename != "second.ar" {
		t.Errorf("trees say %q and %q", first.Filename, second.Filename)
	}
	if len(first.Nodes) != 1 {
		t.Errorf("the first source has %d expressions, want 1", len(first.Nodes))
	}
	if len(second.Nodes) != 2 {
		t.Errorf("the second source has %d expressions, want 2", len(second.Nodes))
	}
}

// A source that does not parse leaves the parser where it was, so the next one still reads.
// This is the REPL: a line that fails is answered and forgotten.
func TestAParserSurvivesASourceThatFails(t *testing.T) {
	p := New(NewParserOptions{})

	if _, err := p.Parse(ParseInput{Filename: "broken.ar", Tokens: tokensOf(t, "1 +;")}); err == nil {
		t.Fatal("a source that does not parse was accepted")
	}

	tree, err := p.Parse(ParseInput{Filename: "fine.ar", Tokens: tokensOf(t, "7;")})
	if err != nil {
		t.Fatalf("after a failure: %v", err)
	}
	if len(tree.Nodes) != 1 {
		t.Errorf("the next source has %d expressions, want 1", len(tree.Nodes))
	}
}

// What `struct` and `as` declare belongs to whoever is compiling, not to the parser: the REPL
// hands the same declarations back every line so a struct declared earlier is still known.
func TestDeclarationsComeFromTheCaller(t *testing.T) {
	p := New(NewParserOptions{})
	declarations := NewDeclarations()

	if _, err := p.Parse(ParseInput{
		Tokens:       tokensOf(t, "struct Point { x, y };"),
		Declarations: declarations,
	}); err != nil {
		t.Fatalf("declaring: %v", err)
	}

	// The next source is a different parse, and it knows the struct because the caller kept it.
	if _, err := p.Parse(ParseInput{
		Tokens:       tokensOf(t, "ident p = Point{10, 20};\np.y;"),
		Declarations: declarations,
	}); err != nil {
		t.Fatalf("using what was declared: %v", err)
	}
}

// And a parse that was handed none knows nothing: two files compiled by the same parser do not
// leak into each other, which is what lets "aurora test" read a source and its test file
// without the test inheriting a name it never declared.
func TestASourceDoesNotSeeWhatAnotherDeclared(t *testing.T) {
	p := New(NewParserOptions{})

	if _, err := p.Parse(ParseInput{
		Tokens:       tokensOf(t, "struct Point { x, y };"),
		Declarations: NewDeclarations(),
	}); err != nil {
		t.Fatalf("declaring: %v", err)
	}

	_, err := p.Parse(ParseInput{Tokens: tokensOf(t, "ident p = Point{10, 20};")})
	if err == nil {
		t.Fatal("a source built a struct that nothing declared in it")
	}
	// A name nothing declared is not a construction at all — it is an identifier followed by
	// a brace — so the parse stops there rather than at the name.
	if !strings.Contains(err.Error(), "unexpected token {") {
		t.Errorf("error = %q, want it to stop at the brace", err)
	}
}
