package parser

import "testing"

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
