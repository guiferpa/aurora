package parser

import (
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/token"
)

// The two instructions that reach what a chain keeps between one transaction and the next.
//
// They are instructions of the machine, not a library: `sstore k v` and `sload k` are the two
// EVM opcodes under the names they have there. What makes them usable is a module written over
// them — the standard library's storage is two scopes doing exactly that — and the wrapper is
// the point. How a key becomes a slot can change inside it without every program that keeps
// something changing with it.
//
// They are written like the other instructions that take what follows them rather than
// parentheses: `sload k` reads as `printd x` does, and `sstore k v` as `head x 1`.

// ParseSload reads `sload k`.
func (p *pr) ParseSload() (ast.Node, error) {
	at, err := p.EatToken(token.SLOAD)
	if err != nil {
		return nil, err
	}
	key, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	return ast.SloadExpression{Key: key, Token: at}, nil
}

// ParseSstore reads `sstore k v`.
//
// The key is read as a whole expression and the value after it, which is the same shape the
// tape operations have. Nothing separates the two, so `sstore a + 1 b` keeps b under a + 1 —
// the reading a person gets from the line, since the key comes first.
func (p *pr) ParseSstore() (ast.Node, error) {
	at, err := p.EatToken(token.SSTORE)
	if err != nil {
		return nil, err
	}
	key, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	value, err := p.ParseExpr()
	if err != nil {
		return nil, err
	}
	return ast.SstoreExpression{Key: key, Value: value, Token: at}, nil
}
