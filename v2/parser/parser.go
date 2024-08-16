package parser

import (
	"errors"
	"fmt"
	"strconv"
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

func (p *pr) getCallee(id IdLiteralNode) (Node, error) {
	params := make([]Node, 0)
	if p.GetLookahead().GetTag().Id != lexer.O_PAREN {
		return id, nil
	}
	p.EatToken(lexer.O_PAREN)
	for p.GetLookahead().GetTag().Id != lexer.C_PAREN {
		expr, err := p.getExpr()
		if err != nil {
			return nil, err
		}
		params = append(params, expr)
		if p.GetLookahead().GetTag().Id == lexer.C_PAREN {
			break
		}
		p.EatToken(lexer.COMMA)
	}
	p.EatToken(lexer.C_PAREN)
	return CalleeLiteralNode{id, params}, nil
}

func (p *pr) getId() (Node, error) {
	tok, err := p.EatToken(lexer.ID)
	if err != nil {
		return nil, err
	}
	id := IdLiteralNode{fmt.Sprintf("%s", tok.GetMatch()), tok}
	return p.getCallee(id)
}

func (p *pr) getNum() (NumberLiteralNode, error) {
	tok, err := p.EatToken(lexer.NUMBER)
	if err != nil {
		return NumberLiteralNode{}, err
	}
	raw := strings.ReplaceAll(string(tok.GetMatch()), "_", "")
	num, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return NumberLiteralNode{}, err
	}
	return NumberLiteralNode{uint64(num), tok}, nil
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
	if lookahead.GetTag().Id == lexer.NUMBER {
		num, err := p.getNum()
		if err != nil {
			return nil, err
		}
		return num, nil
	}
	id, err := p.getId()
	if err != nil {
		return nil, err
	}
	return id, nil
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

func (p *pr) getBoolExpr() (Node, error) {
	left, err := p.getAddExpr()
	if err != nil {
		return nil, err
	}
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.EQUALS {
		op, err := p.EatToken(lexer.EQUALS)
		if err != nil {
			return nil, err
		}
		right, err := p.getBoolExpr()
		if err != nil {
			return nil, err
		}
		return BooleanExpression{left, right, OperationLiteralNode{Value: fmt.Sprintf("%s", op.GetMatch()), Token: op}}, nil
	}
	if lookahead.GetTag().Id == lexer.DIFFERENT {
		op, err := p.EatToken(lexer.DIFFERENT)
		if err != nil {
			return nil, err
		}
		right, err := p.getBoolExpr()
		if err != nil {
			return nil, err
		}
		return BooleanExpression{left, right, OperationLiteralNode{Value: fmt.Sprintf("%s", op.GetMatch()), Token: op}}, nil
	}
	if lookahead.GetTag().Id == lexer.BIGGER {
		op, err := p.EatToken(lexer.BIGGER)
		if err != nil {
			return nil, err
		}
		right, err := p.getBoolExpr()
		if err != nil {
			return nil, err
		}
		return BooleanExpression{left, right, OperationLiteralNode{Value: fmt.Sprintf("%s", op.GetMatch()), Token: op}}, nil
	}
	if lookahead.GetTag().Id == lexer.SMALLER {
		op, err := p.EatToken(lexer.SMALLER)
		if err != nil {
			return nil, err
		}
		right, err := p.getBoolExpr()
		if err != nil {
			return nil, err
		}
		return BooleanExpression{left, right, OperationLiteralNode{Value: fmt.Sprintf("%s", op.GetMatch()), Token: op}}, nil
	}
	return left, nil
}

func (p *pr) getBlockExpr() (Node, error) {
	if _, err := p.EatToken(lexer.O_CUR_BRK); err != nil {
		return nil, err
	}
	stmts, err := p.getStmts(lexer.TagCCurBrk)
	if err != nil {
		return nil, err
	}
	p.EatToken(lexer.C_CUR_BRK)
	return BlockExpressionNode{stmts}, nil
}

func (p *pr) getExpr() (Node, error) {
	if p.GetLookahead().GetTag().Id == lexer.O_CUR_BRK {
		return p.getBlockExpr()
	}
	return p.getBoolExpr()
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
	return IdentStatementNode{fmt.Sprintf("%s", id.GetMatch()), id, expr}, nil
}

func (p *pr) getCallPrint() (Node, error) {
	p.EatToken(lexer.CALL_PRINT)
	p.EatToken(lexer.O_PAREN)
	expr, err := p.getExpr()
	if err != nil {
		return nil, err
	}
	p.EatToken(lexer.C_PAREN)
	return CallPrintStatementNode{expr}, nil
}

func (p *pr) getStmt() (Node, error) {
	lookahead := p.GetLookahead()
	if lookahead.GetTag().Id == lexer.CALL_PRINT {
		return p.getCallPrint()
	}
	if lookahead.GetTag().Id == lexer.IDENT {
		return p.getIdent()
	}
	expr, err := p.getExpr()
	if err != nil {
		return StatementNode{}, err
	}
	return StatementNode{expr}, nil
}

func (p *pr) getStmts(t lexer.Tag) ([]Node, error) {
	stmts := make([]Node, 0)
	for p.GetLookahead().GetTag().Id != t.Id {
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
	stmts, err := p.getStmts(lexer.TagEOF)
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
	module, err := p.getModule()
	if err != nil {
		return AST{}, err
	}
	return AST{module}, nil
}

func New(tokens []lexer.Token) Parser {
	return &pr{cursor: 0, tokens: tokens}
}
