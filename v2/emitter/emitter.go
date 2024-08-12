package emitter

import (
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
		op := n.Operation.Token.GetMatch()
		t := e.genTemp()
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: op, Left: tl, Right: tr})
		return t
	}
	if n, ok := stmt.(parser.NumberLiteralNode); ok {
		t := e.genTemp()
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: []byte{}, Left: n.Value, Right: []byte{}})
		return t
	}
	return make([]byte, 0, 8)
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
