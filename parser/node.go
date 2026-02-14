package parser

import (
	"github.com/guiferpa/aurora/lexer"
)

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

type ParameterLiteralNode struct {
	Expression Node `json:"expression"`
}

func (pln ParameterLiteralNode) Next() Node {
	return pln.Expression
}

type CalleeLiteralNode struct {
	Id     IdLiteralNode          `json:"id"`
	Params []ParameterLiteralNode `json:"params"`
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

type BooleanLiteral struct {
	Value []byte      `json:"value"`
	Token lexer.Token `json:"-"`
}

func (bln BooleanLiteral) Next() Node {
	return nil
}

type NumberLiteralNode struct {
	Value uint64      `json:"value"`
	Token lexer.Token `json:"-"`
}

type ReelLiteralNode struct {
	Value [][]byte    `json:"value"` // Reel as array of tapes: each char is a tape (8 bytes), stored as array of 8-byte arrays
	Token lexer.Token `json:"-"`
}

type VoidLiteralNode struct {
	Token lexer.Token `json:"-"`
}

func (vln VoidLiteralNode) Next() Node {
	return nil
}

func (nln NumberLiteralNode) Next() Node {
	return nil
}

func (rln ReelLiteralNode) Next() Node {
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

type TapeExpression struct {
	Length uint64 `json:"length"`
}

func (TapeExpression) Next() Node {
	return nil
}

type TapeBracketExpression struct {
	Items []Node `json:"items"`
}

func (TapeBracketExpression) Next() Node {
	return nil
}

type PullExpression struct {
	Target Node `json:"target"`
	Item   Node `json:"item"`
}

func (PullExpression) Next() Node {
	return nil
}

type HeadExpression struct {
	Expression Node   `json:"expression"`
	Length     uint64 `json:"length"`
}

func (HeadExpression) Next() Node {
	return nil
}

type TailExpression struct {
	Expression Node   `json:"expression"`
	Length     uint64 `json:"length"`
}

func (TailExpression) Next() Node {
	return nil
}

type PushExpression struct {
	Target Node `json:"target"`
	Item   Node `json:"item"`
}

func (PushExpression) Next() Node {
	return nil
}

type RelativeExpression struct {
	Left      Node                 `json:"left"`
	Right     Node                 `json:"right"`
	Operation OperationLiteralNode `json:"operation"`
}

func (re RelativeExpression) Next() Node {
	return nil
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
	Ref  []byte `json:"id"`
	Body []Node `json:"body"`
}

func (ben BlockExpressionNode) Next() Node {
	return nil
}

// DeferExpressionNode is "defer { ... }". It produces a value that is a pointer to the scope
// (executable later via invocation, e.g. r(1, 2)). No signature or arity.
type DeferExpressionNode struct {
	Ref  []byte `json:"id"`
	Body []Node `json:"body"`
}

func (den DeferExpressionNode) Next() Node {
	return nil
}

type IfExpressionNode struct {
	Test Node                `json:"test"`
	Body []Node              `json:"body"`
	Else *ElseExpressionNode `json:"else"`
}

func (ien IfExpressionNode) Next() Node {
	return nil
}

type ElseExpressionNode struct {
	Body []Node `json:"body"`
}

func (een ElseExpressionNode) Next() Node {
	return nil
}

type ExpressionNode struct {
	Expression Node `json:"expression"`
}

func (en ExpressionNode) Next() Node {
	return en.Expression
}

type PrintStatementNode struct {
	Param Node `json:"parameter"`
}

func (cpsn PrintStatementNode) Next() Node {
	return nil
}

type EchoStatementNode struct {
	Param Node `json:"param"`
}

func (esn EchoStatementNode) Next() Node {
	return nil
}

type ArgumentsExpressionNode struct {
	Nth NumberLiteralNode `json:"nth"`
}

func (aen ArgumentsExpressionNode) Next() Node {
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

type AssertStatementNode struct {
	Condition Node        `json:"condition"`
	Message   Node        `json:"message"`
	Token     lexer.Token `json:"-"`
}

func (asn AssertStatementNode) Next() Node {
	return nil
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

func (mn ModuleNode) Next() Node {
	return nil
}
