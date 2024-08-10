package parser

import "github.com/guiferpa/aurora/lexer"

type Node interface{}

type NumberLiteralNode struct {
	Value int
}

type PrimaryExpressionNode struct {
	Expression Node
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
