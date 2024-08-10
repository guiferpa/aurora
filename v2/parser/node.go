package parser

import "github.com/guiferpa/aurora/lexer"

type Node interface{}

type NumberLiteralNode struct {
	Value int
}

type UnaryExpressionNode struct {
	Expression Node
	Operation  lexer.Token
}

type PrimaryExpressionNode struct {
	Expression Node
}

type ExponentialExpressionNode struct {
	Left      Node
	Right     Node
	Operation lexer.Token
}

type MultiplicativeExpressionNode struct {
	Left      Node
	Right     Node
	Operation lexer.Token
}

type AdditiveExpressionNode struct {
	Left      Node
	Right     Node
	Operation lexer.Token
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
