package parser

import (
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
	"github.com/guiferpa/aurora/wire/token"
)

// What `use storage as s;` brings in, and the two things reached through it.
//
// It is written as an import and is not one: there is no file, nothing is resolved, and
// nothing crosses. What it is to somebody using it is exactly an import — something reached
// under an alias they chose — and writing it that way is what keeps a keyword from being spent
// on it.
//
// The two are `s.set(key, value)` and `s.get(key)`, and both are expressions, because
// everything is. A set returns what it wrote.

// storageMembers is what is reachable through the alias, and whether each one writes.
var storageMembers = map[string]bool{"set": true, "get": false}

// parseStorage reads `s.set(key, value)` or `s.get(key)`.
//
// The alias travels into the tree so a message about a mistake says the word the file actually
// used: somebody who wrote `use storage as db` should read about db.
func (p *pr) parseStorage(alias, symbol string, at token.Token) (ast.Node, error) {
	writes, reachable := storageMembers[symbol]
	if !reachable {
		return nil, token.NewError(at, "storage has no %s at line %d and column %d (it has get, set)",
			symbol, at.GetLine(), at.GetColumn())
	}

	params, err := p.parseStorageParams(alias, symbol, writes, at)
	if err != nil {
		return nil, err
	}

	expression := ast.StorageExpression{Writes: writes, Key: params[0], Alias: alias, Token: at}
	if writes {
		expression.Value = params[1]
	}
	return expression, nil
}

// parseStorageParams reads the parentheses and refuses the wrong number of values inside.
//
// A scope has no signature and is never checked this way, and these are not scopes: they are
// two things the language does, so how many values each takes is the language's to know and
// to say where the mistake was written.
func (p *pr) parseStorageParams(alias, symbol string, writes bool, at token.Token) ([]ast.Node, error) {
	wanted := 1
	if writes {
		wanted = 2
	}

	if _, err := p.EatToken(token.O_PAREN); err != nil {
		return nil, err
	}

	params := make([]ast.Node, 0, wanted)
	for {
		if lookahead := p.GetLookahead(); lookahead != nil && lookahead.GetTag().Id == token.C_PAREN {
			break
		}
		param, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		params = append(params, param)

		if lookahead := p.GetLookahead(); lookahead == nil || lookahead.GetTag().Id != token.COMMA {
			break
		}
		if _, err := p.EatToken(token.COMMA); err != nil {
			return nil, err
		}
	}
	if _, err := p.EatToken(token.C_PAREN); err != nil {
		return nil, err
	}

	if len(params) != wanted {
		return nil, token.NewError(at, "%s.%s takes %s at line %d and column %d, and was given %d",
			alias, symbol, describeStorageArity(writes), at.GetLine(), at.GetColumn(), len(params))
	}
	return params, nil
}

// describeStorageArity says what each of the two takes, in the words somebody writing it would
// use rather than as a number on its own.
func describeStorageArity(writes bool) string {
	if writes {
		return "a key and a value"
	}
	return "a key"
}

// isStorage says whether a specifier is the one that names no file.
func isStorage(specifier string) bool {
	return specifier == module.Storage
}
