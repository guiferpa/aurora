package parser

import "github.com/guiferpa/aurora/lexer"

type Node interface {
	Next() Node
}

type OperationLiteralNode struct {
	Value string      `json:"value"`
	Token lexer.Token `json:"-"`
}

func (oln OperationLiteralNode) Next() Node {
	return nil
}

type CalleeLiteralNode struct {
	Id     IdLiteralNode `json:"id"`
	Params []Node        `json:"params"`
}

func (cln CalleeLiteralNode) Next() Node {
	return nil
}

type IdLiteralNode struct {
	Value string      `json:"value"`
	Token lexer.Token `json:"-"`
}

func (iln IdLiteralNode) Next() Node {
	return nil
}

type NumberLiteralNode struct {
	Value uint64      `json:"value"`
	Token lexer.Token `json:"-"`
}

func (nln NumberLiteralNode) Next() Node {
	return nil
}

type UnaryExpressionNode struct {
	Expression Node                 `json:"expression"`
	Operation  OperationLiteralNode `json:"operation"`
}

func (uen UnaryExpressionNode) Next() Node {
	return uen.Expression
}

type BinaryExpressionNode struct {
	Left      Node                 `json:"left"`
	Right     Node                 `json:"right"`
	Operation OperationLiteralNode `json:"operation"`
}

func (ben BinaryExpressionNode) Next() Node {
	return nil
}

type PrimaryExpressionNode struct {
	Expression Node `json:"expression"`
}

func (pen PrimaryExpressionNode) Next() Node {
	return pen.Expression
}

type BooleanExpression struct {
	Left      Node                 `json:"left"`
	Right     Node                 `json:"right"`
	Operation OperationLiteralNode `json:"operation"`
}

func (be BooleanExpression) Next() Node {
	return nil
}

type BlockExpressionNode struct {
	Statements []Node `json:"statements"`
}

func (ben BlockExpressionNode) Next() Node {
	return nil
}

type ExpressionNode struct {
	Expression Node `json:"expression"`
}

func (en ExpressionNode) Next() Node {
	return en.Expression
}

type CallPrintStatementNode struct {
	Param Node `json:"param"`
}

func (cpsn CallPrintStatementNode) Next() Node {
	return nil
}

type IdentStatementNode struct {
	Id         string      `json:"id"`
	Token      lexer.Token `json:"-"`
	Expression Node        `json:"expression"`
}

func (isn IdentStatementNode) Next() Node {
	return isn.Expression
}

type StatementNode struct {
	Node Node `json:"node"`
}

func (sn StatementNode) Next() Node {
	return sn.Node
}

type ModuleNode struct {
	Name       string `json:"name"`
	Statements []Node `json:"statements"`
}
