package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/guiferpa/aurora/lexer"
)

type Parser interface {
	GetLookahead() lexer.Token
	EatToken(tokenId string) (lexer.Token, error)
	Parse() (AST, error)
}

type pr struct {
	cursor int
	tokens []lexer.Token
}

func (p *pr) getNum() (NumberLiteralNode, error) {
	tok, err := p.EatToken(lexer.NUMBER)
	if err != nil {
		return NumberLiteralNode{}, err
	}
	num := []byte(strings.ReplaceAll(string(tok.GetMatch()), "_", ""))
	fmt.Println(num)
	return NumberLiteralNode{num, tok}, nil
}

func (p *pr) getPriExpr() (Node, error) {
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.O_PAREN {
		if _, err := p.EatToken(lexer.O_PAREN); err != nil {
			return nil, err
		}
		expr, err := p.getExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.EatToken(lexer.C_PAREN); err != nil {
			return nil, err
		}
		return expr, nil
	}
	num, err := p.getNum()
	if err != nil {
		return nil, err
	}
	return PrimaryExpressionNode{num}, nil
}

func (p *pr) getUnaExpr() (Node, error) {
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.SUB {
		op, err := p.EatToken(lexer.SUB)
		if err != nil {
			return nil, err
		}
		expr, err := p.getPriExpr()
		if err != nil {
			return nil, err
		}
		return UnaryExpressionNode{expr, OperationLiteralNode{fmt.Sprintf("%s", op.GetMatch()), op}}, nil
	}
	return p.getPriExpr()
}

func (p *pr) getExpExpr() (Node, error) {
	left, err := p.getUnaExpr()
	if err != nil {
		return nil, err
	}

	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.EXPO {
		op, err := p.EatToken(lexer.EXPO)
		if err != nil {
			return nil, err
		}
		right, err := p.getExpExpr()
		if err != nil {
			return nil, err
		}
		return BinaryExpressionNode{left, right, OperationLiteralNode{fmt.Sprintf("%s", op.GetMatch()), op}}, nil
	}

	return left, nil
}

func (p *pr) getMultExpr() (Node, error) {
	left, err := p.getExpExpr()
	if err != nil {
		return nil, err
	}

	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.MULT {
		op, err := p.EatToken(lexer.MULT)
		if err != nil {
			return nil, err
		}
		right, err := p.getMultExpr()
		if err != nil {
			return nil, err
		}
		return BinaryExpressionNode{left, right, OperationLiteralNode{fmt.Sprintf("%s", op.GetMatch()), op}}, nil
	}
	if lookahead.GetTag().Id == lexer.DIV {
		op, err := p.EatToken(lexer.DIV)
		if err != nil {
			return nil, err
		}
		right, err := p.getMultExpr()
		if err != nil {
			return nil, err
		}
		return BinaryExpressionNode{left, right, OperationLiteralNode{fmt.Sprintf("%s", op.GetMatch()), op}}, nil
	}

	return left, nil
}

func (p *pr) getAddExpr() (Node, error) {
	left, err := p.getMultExpr()
	if err != nil {
		return nil, err
	}

	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.SUM {
		op, err := p.EatToken(lexer.SUM)
		if err != nil {
			return nil, err
		}
		right, err := p.getAddExpr()
		if err != nil {
			return nil, err
		}
		return BinaryExpressionNode{left, right, OperationLiteralNode{fmt.Sprintf("%s", op.GetMatch()), op}}, nil
	}
	if lookahead.GetTag().Id == lexer.SUB {
		op, err := p.EatToken(lexer.SUB)
		if err != nil {
			return nil, err
		}
		right, err := p.getAddExpr()
		if err != nil {
			return nil, err
		}
		return BinaryExpressionNode{left, right, OperationLiteralNode{fmt.Sprintf("%s", op.GetMatch()), op}}, nil
	}

	return left, nil
}

func (p *pr) getExpr() (ExpressionNode, error) {
	add, err := p.getAddExpr()
	if err != nil {
		return ExpressionNode{}, err
	}
	return ExpressionNode{add}, nil
}

func (p *pr) getIdent() (IdentStatementNode, error) {
	if _, err := p.EatToken(lexer.IDENT); err != nil {
		return IdentStatementNode{}, err
	}
	id, err := p.EatToken(lexer.ID)
	if len(id.GetMatch()) == 0 {
		return IdentStatementNode{}, errors.New(fmt.Sprintf("missing identifier name at line: %d, column %d", id.GetLine(), id.GetColumn()))
	}
	if err != nil {
		return IdentStatementNode{}, err
	}
	if _, err := p.EatToken(lexer.ASSIGN); err != nil {
		return IdentStatementNode{}, err
	}
	expr, err := p.getExpr()
	if err != nil {
		return IdentStatementNode{}, err
	}
	return IdentStatementNode{id, expr}, nil
}

func (p *pr) getStmt() (StatementNode, error) {
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.IDENT {
		ident, err := p.getIdent()
		if err != nil {
			return StatementNode{}, err
		}
		return StatementNode{ident}, nil
	}
	expr, err := p.getExpr()
	if err != nil {
		return StatementNode{}, err
	}
	return StatementNode{expr}, nil
}

func (p *pr) getStmts() ([]StatementNode, error) {
	stmts := make([]StatementNode, 0)
	for p.GetLookahead().GetTag().Id != lexer.TagEOF.Id {
		stmt, err := p.getStmt()
		if err != nil {
			return stmts, err
		}
		if _, err := p.EatToken(lexer.SEMICOLON); err != nil {
			return stmts, err
		}
		stmts = append(stmts, stmt)
	}
	return stmts, nil
}

func (p *pr) getModule() (ModuleNode, error) {
	stmts, err := p.getStmts()
	if err != nil {
		return ModuleNode{}, err
	}
	return ModuleNode{"main", stmts}, nil
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
		return currtok, errors.New(fmt.Sprintf("unexpected token %s at line %d and column %d", currtok.GetMatch(), currtok.GetLine(), currtok.GetColumn()))
	}

	p.cursor++

	return currtok, nil
}

func (p *pr) Parse() (AST, error) {
	root, err := p.getModule()
	if err != nil {
		return AST{}, err
	}
	return AST{Root: root}, nil
}

func New(tokens []lexer.Token) Parser {
	return &pr{cursor: 0, tokens: tokens}
}
