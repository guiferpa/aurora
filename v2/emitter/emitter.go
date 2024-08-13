package emitter

import (
	"encoding/binary"
	"fmt"

	"github.com/guiferpa/aurora/parser"
)

type OpCode struct {
	Label     []byte
	Operation []byte
	Left      []byte
	Right     []byte
}

type Emitter interface {
	Emit() []OpCode
}

type emt struct {
	ast     parser.AST
	tmpc    int
	opcodes []OpCode
}

func (e *emt) genTemp() []byte {
	t := []byte(fmt.Sprintf("t%d", e.tmpc))
	e.tmpc++
	return t
}

func (e *emt) fill64Bits(bfs []byte) []byte {
	const size = 8
	bs := make([]byte, size)
	for i := 0; i < len(bfs); i++ {
		bs[(size-len(bfs))+i] = bfs[i]
	}
	return bs
}

func (e *emt) getBytesFromUInt64(v uint64) []byte {
	r := make([]byte, 8)
	binary.BigEndian.PutUint64(r, v)
	return r
}

func (e *emt) emitNode(stmt parser.Node) []byte {
	if n, ok := stmt.(parser.StatementNode); ok {
		return e.emitNode(n.Statement)
	}
	if n, ok := stmt.(parser.PrimaryExpressionNode); ok {
		return e.emitNode(n.Expression)
	}
	if n, ok := stmt.(parser.ExpressionNode); ok {
		return e.emitNode(n.Expression)
	}
	if n, ok := stmt.(parser.UnaryExpressionNode); ok {
		return e.emitNode(n.Expression)
	}
	if n, ok := stmt.(parser.BinaryExpressionNode); ok {
		tl := e.emitNode(n.Left)
		tr := e.emitNode(n.Right)
		op := make([]byte, 8)
		switch fmt.Sprintf("%s", n.Operation.Token.GetMatch()) {
		case "*":
			op = e.fill64Bits([]byte{OpMul})
		case "+":
			op = e.fill64Bits([]byte{OpAdd})
		case "-":
			op = e.fill64Bits([]byte{OpSub})
		case "/":
			op = e.fill64Bits([]byte{OpSub})
		}
		t := e.genTemp()
		e.opcodes = append(e.opcodes, OpCode{Label: e.fill64Bits(t), Operation: op, Left: e.fill64Bits(tl), Right: e.fill64Bits(tr)})
		return t
	}
	if n, ok := stmt.(parser.NumberLiteralNode); ok {
		t := e.genTemp()
		e.opcodes = append(e.opcodes, OpCode{Label: e.fill64Bits(t), Operation: make([]byte, 8), Left: e.getBytesFromUInt64(n.Value), Right: make([]byte, 8)})
		return t
	}
	return make([]byte, 8)
}

func (e *emt) Emit() []OpCode {
	for _, stmt := range e.ast.Root.Statements {
		e.emitNode(stmt.Statement)
	}
	return e.opcodes
}

func New(ast parser.AST) *emt {
	return &emt{ast, 0, make([]OpCode, 0)}
}
