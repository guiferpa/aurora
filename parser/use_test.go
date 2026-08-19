package parser

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/token"
)

// "use" is a keyword again, so it cannot be a name.
//
// It was one when namespaces existed, became an ordinary identifier when they were rolled
// back, and comes back now that modules are designed — which makes this the one breaking
// change the module system asks for. Nothing in the repository was using it as a name, and a
// word that merely starts with it still is one: the lexer has "useful" for that.
func TestUseCannotBeAName(t *testing.T) {
	if _, err := parseSource(t, "ident use = 1;", "main.ar"); err == nil {
		t.Fatal("ident use = 1; parsed, and use is a keyword")
	}
}

// What the line declares: which module, and the name this file reaches it by.
func TestUseDeclaresAModuleUnderAnAlias(t *testing.T) {
	for _, tc := range []struct {
		source    string
		specifier string
		alias     string
	}{
		{"use a/b/c as x;", "a/b/c", "x"},
		{"use math as m;", "math", "m"},
		{"use std/text/case as c;", "std/text/case", "c"},
		// A segment is an ordinary identifier, so it is whatever one may be.
		{"use my_lib/is_true? as t;", "my_lib/is_true?", "t"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			declaration := first[ast.UseDeclaration](t, tc.source)
			if declaration.Specifier != tc.specifier {
				t.Errorf("specifier = %q, want %q", declaration.Specifier, tc.specifier)
			}
			if declaration.Alias != tc.alias {
				t.Errorf("alias = %q, want %q", declaration.Alias, tc.alias)
			}
			if declaration.Token == nil {
				t.Error("the declaration carries no token, so nothing can point at it")
			}
		})
	}
}

// A file names as many modules as it needs, and each alias is written down for whoever reads
// a qualified name later.
func TestUseRecordsEveryAlias(t *testing.T) {
	declarations := NewDeclarations()
	tree, err := parseWithDeclarations(t, "use a/b as x;\nuse c/d as y;", declarations)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if len(tree.Nodes) != 2 {
		t.Fatalf("expected two declarations, got %d", len(tree.Nodes))
	}
	for alias, specifier := range map[string]string{"x": "a/b", "y": "c/d"} {
		if got := declarations.Modules[alias]; got != specifier {
			t.Errorf("alias %s means %q, want %q", alias, got, specifier)
		}
	}
}

// Every way of writing the line wrong, and what it says about it.
func TestUseRefusesWhatIsNotADeclaration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   string
	}{
		{"no alias", "use a/b/c;", "needs a name to be reached by"},
		{"nothing to be called", "use a/b/c as;", "unexpected token"},
		{"no module", "use as x;", "unexpected token"},
		// A path is one word: the same three tokens are a division everywhere else.
		{"spaces in the path", "use a / b as x;", "one word"},
		{"a space before the slash", "use a /b as x;", "one word"},
		{"a space after the slash", "use a/ b as x;", "one word"},
		// The alias belongs to the file, so two of them are a redeclaration.
		{"the alias twice", "use a/b as x;\nuse c/d as x;", "already the alias of a/b"},
		// It belongs to the top, and the top ends at the first thing that is not a use.
		{"after a binding", "ident a = 1;\nuse a/b as x;", "top of the file"},
		{"after a print", "printd 1;\nuse a/b as x;", "top of the file"},
		{"inside a block", "{ use a/b as x; };", "top of the file"},
		{"inside a deferred scope", "ident f = defer { use a/b as x; };", "top of the file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSource(t, tc.source, "main.ar")
			if err == nil {
				t.Fatal("expected a compile error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to say %q", err, tc.want)
			}
			positioned, ok := err.(*token.Error)
			if !ok {
				t.Fatalf("error is %T, want a positioned *token.Error", err)
			}
			if positioned.Line == 0 || positioned.Column == 0 {
				t.Errorf("error has no position: line %d, column %d", positioned.Line, positioned.Column)
			}
		})
	}
}

// A use is still an expression, so what follows one is read as usual.
func TestUseSitsAboveTheRest(t *testing.T) {
	nodes := parse(t, "use a/b as x;\nident n = 1;\nprintd n;")
	if len(nodes) != 3 {
		t.Fatalf("expected three nodes, got %d", len(nodes))
	}
	if _, ok := nodes[0].(ast.UseDeclaration); !ok {
		t.Errorf("the first node is %T, want a use", nodes[0])
	}
}

func parseWithDeclarations(t *testing.T, source string, declarations *Declarations) (ast.AST, error) {
	t.Helper()
	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		return ast.AST{}, err
	}
	return New().Parse(ParseInput{Filename: "main.ar", Tokens: tokens, Declarations: declarations})
}
