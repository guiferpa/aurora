package emitter

import (
	"bytes"
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
	Emit() ([]OpCode, error)
}

type emt struct {
	ast     parser.AST
	tmpc    int
	opcodes []OpCode
}

func (e *emt) genLabel() []byte {
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
		return e.emitNode(n.Node)
	}
	if n, ok := stmt.(parser.IdentStatementNode); ok {
		texpr := e.emitNode(n.Expression)
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: e.fill64Bits(t), Operation: e.fill64Bits([]byte{OpPin}), Left: e.fill64Bits(n.Token.GetMatch()), Right: e.fill64Bits(texpr)})
	}
	if n, ok := stmt.(parser.UnaryExpressionNode); ok {
		return e.emitNode(n.Expression)
	}
	if n, ok := stmt.(parser.BlockExpressionNode); ok {
		t := e.fill64Bits(e.genLabel())
		op := e.fill64Bits([]byte{OpOBl})
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: op, Left: e.fill64Bits([]byte{}), Right: e.fill64Bits([]byte{})})

		for _, stmt := range n.Statements {
			e.emitNode(stmt)
		}

		latest := e.opcodes[len(e.opcodes)-1]
		isEmpty := bytes.Compare(latest.Label, t) == 0
		op = e.fill64Bits([]byte{OpCBl})
		t = e.fill64Bits(e.genLabel())
		if isEmpty {
			e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: op, Left: e.fill64Bits([]byte{}), Right: e.fill64Bits([]byte{})})
			return t
		}
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: op, Left: e.fill64Bits([]byte{}), Right: e.fill64Bits([]byte{})})

		t = e.fill64Bits(e.genLabel())
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: e.fill64Bits([]byte{OpLab}), Left: latest.Label, Right: e.fill64Bits([]byte{})})
		return t
	}
	if n, ok := stmt.(parser.BooleanExpression); ok {
		tl := e.emitNode(n.Left)
		tr := e.emitNode(n.Right)
		op := make([]byte, 8)
		switch fmt.Sprintf("%s", n.Operation.Token.GetMatch()) {
		case "equals":
			op = e.fill64Bits([]byte{OpEqu})
		case "different":
			op = e.fill64Bits([]byte{OpDif})
		case "bigger":
			op = e.fill64Bits([]byte{OpBig})
		case "smaller":
			op = e.fill64Bits([]byte{OpSma})
		}
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: e.fill64Bits(t), Operation: op, Left: e.fill64Bits(tl), Right: e.fill64Bits(tr)})
		return t
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
		case "^":
			op = e.fill64Bits([]byte{OpExp})
		}
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: e.fill64Bits(t), Operation: op, Left: e.fill64Bits(tl), Right: e.fill64Bits(tr)})
		return t
	}
	if n, ok := stmt.(parser.NumberLiteralNode); ok {
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: e.fill64Bits(t), Operation: e.fill64Bits([]byte{OpLab}), Left: e.getBytesFromUInt64(n.Value), Right: make([]byte, 8)})
		return t
	}
	if n, ok := stmt.(parser.IdLiteralNode); ok {
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: e.fill64Bits(t), Operation: e.fill64Bits([]byte{OpGet}), Left: e.fill64Bits(n.Token.GetMatch()), Right: e.fill64Bits([]byte{})})
		return t
	}
	return make([]byte, 8)
}

func (e *emt) Emit() ([]OpCode, error) {
	for _, stmt := range e.ast.Module.Statements {
		e.emitNode(stmt.Node)
	}
	return e.opcodes, nil
}

func New(ast parser.AST) *emt {
	return &emt{ast, 0, make([]OpCode, 0)}
}
