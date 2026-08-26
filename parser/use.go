package parser

import (
	"strings"

	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/token"
)

// What `use` declares, and the rules that come with it.
//
// A module is a file, and the name of a module is the path it has from the source root:
// `use a/b/c as x;` reads a/b/c.ar. The alias is how everything inside is reached, and it is
// mandatory — reading `x.add(1, 2)` says where add lives without anyone scrolling anywhere.
//
// Nothing here resolves anything. The parser writes down which alias means which module and
// refuses what is wrong about the line itself; finding the file, ordering it and checking
// that a symbol exists in it are other people's jobs.

// ParseUse reads `use a/b/c as x;`.
func (p *pr) ParseUse() (ast.Node, error) {
	tok, err := p.EatToken(token.USE)
	if err != nil {
		return nil, err
	}
	// The rule is the reader's: what a file depends on is known before anything happens, so
	// nobody has to scroll to find out.
	if !p.useAllowed {
		return nil, token.NewError(tok, "use belongs to the top of the file at line %d and column %d: put it above everything else",
			tok.GetLine(), tok.GetColumn())
	}

	specifier, err := p.parseSpecifier()
	if err != nil {
		return nil, err
	}

	// The alias is what the rest of the file reaches the module by, so leaving it out is the
	// one mistake worth naming: everything else here is a token in the wrong place.
	if lookahead := p.GetLookahead(); lookahead != nil && lookahead.GetTag().Id != token.AS {
		return nil, token.NewError(lookahead, "an import needs a name to be reached by at line %d and column %d: use %s as x",
			lookahead.GetLine(), lookahead.GetColumn(), specifier)
	}
	if _, err := p.EatToken(token.AS); err != nil {
		return nil, err
	}
	name, err := p.EatToken(token.ID)
	if err != nil {
		return nil, err
	}

	alias := string(name.GetMatch())
	if declared, ok := p.declarations.Modules[alias]; ok {
		return nil, token.NewError(name, "%s is already the alias of %s at line %d and column %d",
			alias, declared, name.GetLine(), name.GetColumn())
	}
	p.declarations.Modules[alias] = specifier
	// Storage has no file and offers no shapes, so there is nothing to import: what the alias
	// reaches is written into the language rather than read from anywhere.
	if !isStorage(specifier) {
		p.declarations.Import(specifier, p.imports[specifier])
	}

	return ast.UseDeclaration{Specifier: specifier, Alias: alias, Token: tok}, nil
}

// parseSpecifier reads `a/b/c` as one path.
//
// The segments are ordinary identifiers with the division sign between them, which is all the
// lexer needs to know — a path needs no token of its own. What it does need is to be written
// as one word: see glued.
func (p *pr) parseSpecifier() (string, error) {
	first, err := p.EatToken(token.ID)
	if err != nil {
		return "", err
	}

	segments := []string{string(first.GetMatch())}
	previous := first
	for p.GetLookahead() != nil && p.GetLookahead().GetTag().Id == token.DIV {
		slash, err := p.EatToken(token.DIV)
		if err != nil {
			return "", err
		}
		if err := glued(previous, slash); err != nil {
			return "", err
		}
		segment, err := p.EatToken(token.ID)
		if err != nil {
			return "", err
		}
		if err := glued(slash, segment); err != nil {
			return "", err
		}
		segments = append(segments, string(segment.GetMatch()))
		previous = segment
	}
	return strings.Join(segments, "/"), nil
}

// glued refuses a space inside a path.
//
// `a / b` is a division everywhere else in the language, and reading the same three tokens as
// a path here would make them mean two things depending on the line they sit in. A cursor is
// a byte offset, so two tokens are one word when the second starts where the first ended.
func glued(left, right token.Token) error {
	if right.GetCursor() == left.GetCursor()+len(left.GetMatch()) {
		return nil
	}
	return token.NewError(right, "a module path is written as one word at line %d and column %d: a/b/c, not a / b / c",
		right.GetLine(), right.GetColumn())
}
