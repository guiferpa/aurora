package parser

import (
	"github.com/guiferpa/aurora/lexer"
)

type Node interface {
	Next() Node
}

type OperationLiteral struct {
	Value string      `json:"value"`
	Token lexer.Token `json:"-"`
}

func (oln OperationLiteral) Next() Node {
	return nil
}

type ParameterLiteral struct {
	Expression Node `json:"expression"`
}

func (pln ParameterLiteral) Next() Node {
	return pln.Expression
}

type CalleeLiteral struct {
	Id     IdentifierLiteral  `json:"identifier"`
	Params []ParameterLiteral `json:"parameters"`
}

func (cln CalleeLiteral) Next() Node {
	return nil
}

type IdentifierLiteral struct {
	Value string      `json:"value"`
	Token lexer.Token `json:"-"`
}

func (iln IdentifierLiteral) Next() Node {
	return nil
}

type BooleanLiteral struct {
	Value []byte      `json:"value"`
	Token lexer.Token `json:"-"`
}

func (bln BooleanLiteral) Next() Node {
	return nil
}

type NumberLiteral struct {
	Value uint64      `json:"value"`
	Token lexer.Token `json:"-"`
}

type ReelLiteral struct {
	Value [][]byte    `json:"value"` // Reel as array of tapes: each char is a tape (8 bytes), stored as array of 8-byte arrays
	Token lexer.Token `json:"-"`
}

type VoidLiteral struct {
	Token lexer.Token `json:"-"`
}

func (vln VoidLiteral) Next() Node {
	return nil
}

func (nln NumberLiteral) Next() Node {
	return nil
}

func (rln ReelLiteral) Next() Node {
	return nil
}

type UnaryExpression struct {
	Expression Node             `json:"expression"`
	Operation  OperationLiteral `json:"operation"`
}

func (uen UnaryExpression) Next() Node {
	return uen.Expression
}

type BinaryExpression struct {
	Left      Node             `json:"left"`
	Right     Node             `json:"right"`
	Operation OperationLiteral `json:"operation"`
}

func (ben BinaryExpression) Next() Node {
	return nil
}

type PrimaryExpression struct {
	Expression Node `json:"expression"`
}

func (pen PrimaryExpression) Next() Node {
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
	Left      Node             `json:"left"`
	Right     Node             `json:"right"`
	Operation OperationLiteral `json:"operation"`
}

func (re RelativeExpression) Next() Node {
	return nil
}

type BooleanExpression struct {
	Left      Node             `json:"left"`
	Right     Node             `json:"right"`
	Operation OperationLiteral `json:"operation"`
}

func (be BooleanExpression) Next() Node {
	return nil
}

type BlockExpression struct {
	Body []Node `json:"body"`
}

func (ben BlockExpression) Next() Node {
	return nil
}

// DeferExpression is "defer { ... }". It produces a value that is a pointer to the scope
// (executable later via invocation, e.g. r(1, 2)). No signature or arity.
// Block is the body of the defer; it is a BlockExpression so the emitter can treat it
// as a normal scope (BeginScope + body + Return) without duplicating scope logic.
type DeferExpression struct {
	Block BlockExpression `json:"block"`
}

func (den DeferExpression) Next() Node {
	return nil
}

type IfExpression struct {
	Test Node               `json:"test"`
	Body []Node             `json:"body"`
	Else *ElseExpression   `json:"else"`
}

func (ien IfExpression) Next() Node {
	return nil
}

type ElseExpression struct {
	Body []Node `json:"body"`
}

func (een ElseExpression) Next() Node {
	return nil
}

type Expression struct {
	Expression Node `json:"expression"`
}

func (en Expression) Next() Node {
	return en.Expression
}

type PrintStatement struct {
	Param Node `json:"parameter"`
}

func (cpsn PrintStatement) Next() Node {
	return nil
}

type EchoStatement struct {
	Param Node `json:"param"`
}

func (esn EchoStatement) Next() Node {
	return nil
}

type ArgumentsExpression struct {
	Nth NumberLiteral `json:"nth"`
}

func (aen ArgumentsExpression) Next() Node {
	return nil
}

type IdentStatement struct {
	Id         string      `json:"id"`
	Token      lexer.Token `json:"-"`
	Expression Node        `json:"expression"`
}

func (isn IdentStatement) Next() Node {
	return isn.Expression
}

type AssertStatement struct {
	Condition Node        `json:"condition"`
	Message   Node        `json:"message"`
	Token     lexer.Token `json:"-"`
}

func (asn AssertStatement) Next() Node {
	return nil
}

type Statement struct {
	Node Node `json:"node"`
}

func (sn Statement) Next() Node {
	return sn.Node
}

type Module struct {
	Name       string `json:"name"`
	Statements []Node `json:"statements"`
}

func (mn Module) Next() Node {
	return nil
}
