package parser

import (
	"errors"
	"fmt"

	"github.com/guiferpa/aurora/lexer"
)

type Parser interface {
	GetLookahead() lexer.Token
	EatToken(tokenId string) (lexer.Token, error)
	Parse() AST
}

type pr struct {
	cursor    int
	lookahead lexer.Token
	tokens    []lexer.Token
}

func (p *pr) getExpr() Node {
	return n{}
}

func (p *pr) getIdent() Node {
	return n{}
}

func (p *pr) getStmt() Node {
	return n{}
}

func (p *pr) getStmts(tokens []lexer.Token) []Node {
	return []Node{}
}

func (p *pr) getModule() Node {
	return n{}
}

func (p *pr) GetLookahead() lexer.Token {
	return p.lookahead
}

func (p *pr) EatToken(tokenId string) (lexer.Token, error) {
	currtok := p.lookahead

	if tokenId != currtok.GetTag().Id {
		return nil, errors.New(fmt.Sprintf("unexpected token %s at line %d and column %d", currtok.GetMatch(), currtok.GetLine(), currtok.GetColumn()))
	}

	p.lookahead = p.tokens[p.cursor]
	p.cursor++

	return currtok, nil
}

func (p *pr) Parse() AST {
	if len(p.tokens) == 0 {
		return AST{Root: make([]Node, 0)}
	}
	p.lookahead = p.tokens[p.cursor]
	p.cursor++
	return AST{Root: p.getModule()}
}

func New(tokens []lexer.Token) Parser {
	return &pr{cursor: 0, tokens: tokens, lookahead: nil}
}
