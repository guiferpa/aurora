package parser

import (
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
	"github.com/guiferpa/aurora/wire/token"
)

// How a name is written down, which is not always how it was typed.
//
// Inside a module every identifier carries the module in front of it: in a/b/c, `ident base`
// is written a/b/c.base. It is a constant renaming of the file's own names, so nothing here
// needs to know what is a binding and what is a mention — every one of them is renamed the
// same way, and they go on finding each other.
//
// Two things come out of it. Two modules binding the same word are two different names, which
// is what lets them share one environ at run time. And the name says which module it belongs
// to, which is what the second hop of a lookup follows: see docs/module_system_design.md.

// name is what an identifier is called in the program. The file somebody asked to run has no
// module, and its names are written as they were typed.
func (p *pr) name(text string) string {
	return module.Qualify(module.ID(p.module), text)
}

// typed is what an identifier says in the file, which is what a shape name and an alias are
// looked up by: neither ever reaches an instruction, and neither leaves the file.
func typed(id ast.IdentifierLiteral) string {
	return string(id.Token.GetMatch())
}

// parseMember reads what comes after the dot when the head is a module alias.
//
// What comes out is an ordinary identifier under the name the module itself writes, so
// everything downstream — a call, a load, the environ — goes on treating it as any other
// name. The alias is gone by then: it belonged to this file and to nothing else.
func (p *pr) parseMember(specifier, symbol string, at token.Token) (ast.Node, error) {
	// Storage names no file, so nothing about it is looked up: what follows the dot is one of
	// two things the language does, and the alias is only how this file spells it.
	if isStorage(specifier) {
		return p.parseStorage(p.aliasOf(specifier), symbol, at)
	}
	// A shape of that module is not a value there any more than a local one is here: it is
	// built, or it names what a value is read as, and nothing else.
	shape := module.Qualify(module.ID(specifier), symbol)
	if _, declared := p.declarations.Shapes[shape]; declared {
		if p.GetLookahead() != nil && p.GetLookahead().GetTag().Id == token.O_CUR_BRK {
			return p.parseShapeValue(shape, symbol, at)
		}
		return nil, token.NewError(at, "%s is a shape of module %s at line %d and column %d: build a value with %s{...}",
			symbol, specifier, at.GetLine(), at.GetColumn(), symbol)
	}

	p.references = append(p.references, ast.Reference{Module: specifier, Symbol: symbol, Token: at})

	qualified := ast.IdentifierLiteral{Value: module.Qualify(module.ID(specifier), symbol), Token: at}
	if p.GetLookahead() != nil && p.GetLookahead().GetTag().Id == token.O_PAREN {
		return p.ParseCallee(qualified)
	}
	return qualified, nil
}

// aliasOf answers what this file calls a specifier, for a message somebody reads. A specifier
// with no alias here cannot be reached at all, so the specifier itself is the honest fallback.
func (p *pr) aliasOf(specifier string) string {
	for alias, named := range p.declarations.Modules {
		if named == specifier {
			return alias
		}
	}
	return specifier
}
