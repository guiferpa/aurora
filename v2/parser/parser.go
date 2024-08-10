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
	cursor int
	tokens []lexer.Token
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

func (p *pr) getStmts() []Node {
	return []Node{}
}

func (p *pr) getModule() Node {
	return n{}
}

func (p *pr) GetLookahead() lexer.Token {
	if p.cursor < len(p.tokens) {
		return p.tokens[p.cursor]
	}
	return nil
}

func (p *pr) EatToken(tokenId string) (lexer.Token, error) {
	currtok := p.GetLookahead()

	if currtok == nil {
		return nil, nil
	}

	if tokenId != currtok.GetTag().Id {
		return nil, errors.New(fmt.Sprintf("unexpected token %s at line %d and column %d", currtok.GetMatch(), currtok.GetLine(), currtok.GetColumn()))
	}

	p.cursor++

	return currtok, nil
}

func (p *pr) Parse() AST {
	return AST{Root: p.getModule()}
}

func New(tokens []lexer.Token) Parser {
	return &pr{cursor: 0, tokens: tokens}
}
