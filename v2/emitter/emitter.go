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
	t := []byte(fmt.Sprintf("%dt", e.tmpc))
	e.tmpc++
	return t
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
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: []byte{OpPin}, Left: n.Token.GetMatch(), Right: texpr})
	}
	if n, ok := stmt.(parser.FuncExpressionNode); ok {
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: []byte{OpFun}, Left: []byte(n.Ref), Right: []byte{}})
		t = e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: []byte{OpLab}, Left: []byte(n.Ref), Right: []byte{}})
		return t
	}
	if n, ok := stmt.(parser.UnaryExpressionNode); ok {
		return e.emitNode(n.Expression)
	}
	if n, ok := stmt.(parser.BlockExpressionNode); ok {
		t := e.genLabel()
		op := []byte{OpOBl}
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: op, Left: []byte{}, Right: ([]byte{})})

		for _, stmt := range n.Statements {
			e.emitNode(stmt)
		}

		latest := e.opcodes[len(e.opcodes)-1]
		isEmpty := bytes.Compare(latest.Label, t) == 0
		op = ([]byte{OpCBl})
		t = (e.genLabel())
		if isEmpty {
			e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: op, Left: ([]byte{}), Right: ([]byte{})})
			return t
		}
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: op, Left: ([]byte{}), Right: ([]byte{})})

		t = (e.genLabel())
		e.opcodes = append(e.opcodes, OpCode{Label: t, Operation: ([]byte{OpLab}), Left: latest.Label, Right: ([]byte{})})
		return t
	}
	if n, ok := stmt.(parser.BooleanExpression); ok {
		tl := e.emitNode(n.Left)
		tr := e.emitNode(n.Right)
		op := make([]byte, 0)
		switch fmt.Sprintf("%s", n.Operation.Token.GetMatch()) {
		case "equals":
			op = ([]byte{OpEqu})
		case "different":
			op = ([]byte{OpDif})
		case "bigger":
			op = ([]byte{OpBig})
		case "smaller":
			op = ([]byte{OpSma})
		}
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: (t), Operation: op, Left: (tl), Right: (tr)})
		return t
	}
	if n, ok := stmt.(parser.BinaryExpressionNode); ok {
		tl := e.emitNode(n.Left)
		tr := e.emitNode(n.Right)
		op := make([]byte, 0)
		switch fmt.Sprintf("%s", n.Operation.Token.GetMatch()) {
		case "*":
			op = ([]byte{OpMul})
		case "+":
			op = ([]byte{OpAdd})
		case "-":
			op = ([]byte{OpSub})
		case "/":
			op = ([]byte{OpSub})
		case "^":
			op = ([]byte{OpExp})
		}
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: (t), Operation: op, Left: (tl), Right: (tr)})
		return t
	}
	if n, ok := stmt.(parser.NumberLiteralNode); ok {
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: (t), Operation: ([]byte{OpLab}), Left: e.getBytesFromUInt64(n.Value), Right: make([]byte, 0)})
		return t
	}
	if n, ok := stmt.(parser.IdLiteralNode); ok {
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: (t), Operation: ([]byte{OpGet}), Left: (n.Token.GetMatch()), Right: ([]byte{})})
		return t
	}
	if n, ok := stmt.(parser.CalleeLiteralNode); ok {
		for _, p := range n.Params {
			pn := e.emitNode(p)
			t := e.genLabel()
			e.opcodes = append(e.opcodes, OpCode{Label: (t), Operation: ([]byte{OpPar}), Left: (pn), Right: ([]byte{})})
		}
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: (t), Operation: ([]byte{OpCal}), Left: (n.Id.Token.GetMatch()), Right: ([]byte{})})
		return t
	}
	if n, ok := stmt.(parser.CallPrintStatementNode); ok {
		tl := e.emitNode(n.Param)
		t := e.genLabel()
		e.opcodes = append(e.opcodes, OpCode{Label: (t), Operation: ([]byte{OpPrt}), Left: (tl), Right: ([]byte{})})
		return t
	}
	return make([]byte, 8)
}

func (e *emt) Emit() ([]OpCode, error) {
	for _, stmt := range e.ast.Module.Statements {
		e.emitNode(stmt)
	}
	return e.opcodes, nil
}

func New(ast parser.AST) *emt {
	return &emt{ast, 0, make([]OpCode, 0)}
}
