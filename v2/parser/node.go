package parser

import "github.com/guiferpa/aurora/lexer"

type Node interface {
	Next() Node
}

type OperationLiteralNode struct {
	Value string
	Token lexer.Token
}

func (oln OperationLiteralNode) Next() Node {
	return nil
}

type IdLiteralNode struct {
	Value string
	Token lexer.Token
}

func (iln IdLiteralNode) Next() Node {
	return nil
}

type NumberLiteralNode struct {
	Value uint64
	Token lexer.Token
}

func (nln NumberLiteralNode) Next() Node {
	return nil
}

type UnaryExpressionNode struct {
	Expression Node
	Operation  OperationLiteralNode
}

func (uen UnaryExpressionNode) Next() Node {
	return uen.Expression
}

type BinaryExpressionNode struct {
	Left      Node
	Right     Node
	Operation OperationLiteralNode
}

func (ben BinaryExpressionNode) Next() Node {
	return nil
}

type PrimaryExpressionNode struct {
	Expression Node
}

func (pen PrimaryExpressionNode) Next() Node {
	return pen.Expression
}

type ExpressionNode struct {
	Expression Node
}

func (en ExpressionNode) Next() Node {
	return en.Expression
}

type IdentStatementNode struct {
	Name       lexer.Token
	Expression Node
}

func (isn IdentStatementNode) Next() Node {
	return isn.Expression
}

type StatementNode struct {
	Statement Node
}

func (sn StatementNode) Next() Node {
	return sn.Statement
}

type ModuleNode struct {
	Name       string
	Statements []StatementNode
}
