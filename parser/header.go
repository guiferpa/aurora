package parser

import (
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/token"
)

// Reading the top of a file without parsing it.
//
// What a file imports is the first thing in it, and two callers need it before a parse can
// happen: the resolver, which has to read a module's dependencies before reading the module,
// and the editor, which is asked what is inside a module while the document is too broken to
// parse. Both ask here.

// scanUses reads every `use a/b/c as x;` out of a token chain, with the token of each so
// whatever goes wrong with one can be underlined where it was written.
func ScanUses(tokens []token.Token) []ast.UseDeclaration {
	found := make([]ast.UseDeclaration, 0)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].GetTag().Id != token.USE {
			continue
		}
		if alias, specifier, next := readUse(tokens, i); alias != "" {
			found = append(found, ast.UseDeclaration{Specifier: specifier, Alias: alias, Token: tokens[i]})
			i = next
		}
	}
	return found
}

// readUse reads one declaration starting at the use token: the path it names, and the alias
// it is reached by. A half-written line returns nothing, which is the common case while
// somebody is typing one.
func readUse(tokens []token.Token, i int) (string, string, int) {
	specifier := ""
	for j := i + 1; j < len(tokens); j++ {
		switch tokens[j].GetTag().Id {
		case token.ID:
			specifier += string(tokens[j].GetMatch())
		case token.DIV:
			specifier += "/"
		case token.AS:
			if specifier == "" || j+1 >= len(tokens) || tokens[j+1].GetTag().Id != token.ID {
				return "", "", j
			}
			return string(tokens[j+1].GetMatch()), specifier, j + 1
		default:
			return "", "", j
		}
	}
	return "", "", len(tokens) - 1
}
