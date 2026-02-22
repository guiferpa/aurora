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
	Value     string      `json:"value"`     // symbol name (e.g. "open_file")
	Namespace string      `json:"namespace"` // optional path segments (e.g. "std::fs::io"); empty = simple identifier
	Token     lexer.Token `json:"-"`
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

func (nln NumberLiteral) Next() Node {
	return nil
}

type NothingLiteral struct {
	Token lexer.Token `json:"-"`
}

func (nln NothingLiteral) Next() Node {
	return nil
}

type ReelLiteral struct {
	Value [][]byte    `json:"value"` // Reel as array of tapes: each char is a tape (8 bytes), stored as array of 8-byte arrays
	Token lexer.Token `json:"-"`
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
	Test Node            `json:"test"`
	Body []Node          `json:"body"`
	Else *ElseExpression `json:"else"`
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

type IdentLiteral struct {
	Id    string      `json:"id"`
	Token lexer.Token `json:"-"`
	Value Node        `json:"value"`
}

func (isn IdentLiteral) Next() Node {
	return isn.Value
}

type AssertStatement struct {
	Condition Node        `json:"condition"`
	Message   Node        `json:"message"`
	Token     lexer.Token `json:"-"`
}

func (asn AssertStatement) Next() Node {
	return nil
}

// UseDeclaration is "use path::to::ns as alias;". Path is the namespace path segments; Alias is the local name.
// Resolution and linking are implicit; this node only records the alias for the rest of the compiler.
type UseDeclaration struct {
	Namespace string      `json:"namespace"` // e.g. "std::fs::io"
	Alias     string      `json:"alias"`     // e.g. "io" (alias for "std::fs::io")
	Token     lexer.Token `json:"-"`
}

func (ud UseDeclaration) Next() Node {
	return nil
}

// Module is the top-level AST node. Aurora is expression-only: Expressions is
// the sequence of expressions at top level (e.g. NothingLiteral, IdentStatement,
// BlockExpression, IfExpression).
type Module struct {
	Name        string `json:"name"`
	Expressions []Node `json:"expressions"`
}

func (mn Module) Next() Node {
	return nil
}
