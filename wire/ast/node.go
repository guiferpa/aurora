package ast

import (
	"github.com/guiferpa/aurora/wire/token"
)

type Node interface {
	Next() Node
}

type OperationLiteral struct {
	Value string      `json:"value"`
	Token token.Token `json:"-"`
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
	Token token.Token `json:"-"`
}

func (iln IdentifierLiteral) Next() Node {
	return nil
}

type BooleanLiteral struct {
	Value []byte      `json:"value"`
	Token token.Token `json:"-"`
}

func (bln BooleanLiteral) Next() Node {
	return nil
}

type NumberLiteral struct {
	Value uint64      `json:"value"`
	Token token.Token `json:"-"`
}

func (nln NumberLiteral) Next() Node {
	return nil
}

// TextLiteral is `"text"`: the bytes of the text, packed into one tape.
//
// It is one more way of writing a tape, next to `1`, `0x2a`, `[1, 2]` and `true` — not a
// kind of value of its own.
//
// Text used to be a reel — one tape per character, so "Gui" was three tapes and 24 bytes at
// the default width. That was a way of holding more than a tape could, and it cost
// tape_size - 1 bytes of zero for every character. Text is now the bytes it is, right
// aligned like every other value, so "Gui" is three bytes and "a" is 97, the same tape the
// number 97 is.
type TextLiteral struct {
	Value []byte      `json:"value"`
	Token token.Token `json:"-"`
}

func (tln TextLiteral) Next() Node {
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

// PrintFormat is how a print builtin reads the tape it is given.
type PrintFormat string

const (
	PrintBytes   PrintFormat = "bytes"   // printb
	PrintChars   PrintFormat = "chars"   // printc
	PrintDecimal PrintFormat = "decimal" // printd
)

// PrintStatement is one of the print builtins. They differ only in how the value is read,
// so they are one node carrying which reading was asked for.
type PrintStatement struct {
	Format PrintFormat `json:"format"`
	Param  Node        `json:"parameter"`
}

func (psn PrintStatement) Next() Node {
	return nil
}

// FeedExpression is "feed(n)": it reads the nth value applied to the running scope.
type FeedExpression struct {
	Nth NumberLiteral `json:"nth"`
}

func (fen FeedExpression) Next() Node {
	return nil
}

type IdentLiteral struct {
	Id    string      `json:"id"`
	Token token.Token `json:"-"`
	Value Node        `json:"value"`
}

func (isn IdentLiteral) Next() Node {
	return isn.Value
}

// AssertStatement is `assert(condition, "message")`.
//
// The message is a literal held as text, not as a value: it is written for whoever reads
// the result of a test, the same way a struct's field names are written for whoever reads
// the source. Keeping it out of the values is also what lets it be longer than a tape,
// which a message usually is.
type AssertStatement struct {
	Condition Node        `json:"condition"`
	Message   string      `json:"message"`
	Token     token.Token `json:"-"`
}

func (asn AssertStatement) Next() Node {
	return nil
}

// AST is the top-level node: Aurora is expression-only, so a parsed file is the sequence
// of expressions it holds. The unit of compilation is the file.
type AST struct {
	Filename string `json:"filename"`
	Nodes    []Node `json:"nodes"`
}

func (a AST) Next() Node {
	return nil
}

// StructDeclaration names the fields of a run of tapes: `struct Point { x, y };`.
//
// It is a directive for whoever writes the source, not a value. The compiler reads it to
// turn a field name into an index, to report a mistake where it was written, and to feed
// the language server — and then it is gone. Nothing about it reaches the IR: the flow is
// static, the fields are positional and every one of them is exactly a tape wide.
type StructDeclaration struct {
	Name   string      `json:"name"`
	Fields []string    `json:"fields"`
	Token  token.Token `json:"-"`
}

func (sdn StructDeclaration) Next() Node {
	return nil
}

// StructLiteral builds the run: `Point{10, 20}` is two tapes, one per field.
type StructLiteral struct {
	Name   string      `json:"name"`
	Values []Node      `json:"values"`
	Token  token.Token `json:"-"`
}

func (sln StructLiteral) Next() Node {
	return nil
}

// FieldExpression reads one tape out of a run. The index is resolved while parsing, from
// the shape of the value, so nothing about the field's name survives here.
type FieldExpression struct {
	Expression Node        `json:"expression"`
	Index      uint64      `json:"index"`
	Field      string      `json:"field"` // kept for the language server, never emitted
	Token      token.Token `json:"-"`
}

func (fen FieldExpression) Next() Node {
	return fen.Expression
}

// ShapedExpression is `expr as Point`: it says how to read a value the compiler cannot
// trace back to a construction. It has no effect of its own — the emitter emits what is
// inside it and drops the name.
type ShapedExpression struct {
	Expression Node        `json:"expression"`
	Struct     string      `json:"struct"`
	Token      token.Token `json:"-"`
}

func (sen ShapedExpression) Next() Node {
	return sen.Expression
}
