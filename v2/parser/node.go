package parser

import "github.com/guiferpa/aurora/lexer"

type Node interface{}

type OperationLiteralNode struct {
	Value string
	Token lexer.Token
}

type NumberLiteralNode struct {
	Value int
	Token lexer.Token
}

type UnaryExpressionNode struct {
	Expression Node
	Operation  OperationLiteralNode
}

type BinaryExpressionNode struct {
	Left      Node
	Right     Node
	Operation OperationLiteralNode
}

type PrimaryExpressionNode struct {
	Expression Node
}

type ExpressionNode struct {
	Expression Node
}

type IdentStatementNode struct {
	Name       lexer.Token
	Expression Node
}

type StatementNode struct {
	Statement Node
}

type ModuleNode struct {
	Name       string
	Statements []StatementNode
}
