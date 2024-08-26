package emitter

import (
	"bytes"
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/parser"
)

type Emitter interface {
	Emit() ([]Instruction, error)
}

type emt struct {
	ast   parser.AST
	tmpc  int
	insts []Instruction
}

func (e *emt) generateLabel() []byte {
	t := []byte(fmt.Sprintf("%dt", e.tmpc))
	e.tmpc++
	return t
}

func (e *emt) emitInstruction(stmt parser.Node) []byte {
	if n, ok := stmt.(parser.StatementNode); ok {
		return e.emitInstruction(n.Node)
	}
	if n, ok := stmt.(parser.IdentStatementNode); ok {
		ll := n.Token.GetMatch()
		lr := e.emitInstruction(n.Expression)
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpIdentify, ll, lr))
	}
	if n, ok := stmt.(parser.FuncExpressionNode); ok {
		cins := e.insts
		e.insts = make([]Instruction, 0)
		for _, ins := range n.Body {
			e.emitInstruction(ins)
		}
		var length uint64 = uint64(len(n.Arity)) + uint64(len(e.insts)) + 1 // Length of function
		e.insts = cins

		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpBeginFunc, []byte(n.Ref), byteutil.FromUint64(length)))

		for i, a := range n.Arity {
			l := e.generateLabel()
			e.insts = append(e.insts, NewInstruction(l, OpLoadParam, a.Token.GetMatch(), byteutil.FromUint64(uint64(i))))
		}

		for _, ins := range n.Body {
			l = e.emitInstruction(ins)
		}

		rl := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(rl, OpReturn, l, nil))

		l = e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpSave, []byte(n.Ref), nil))

		return l
	}
	if n, ok := stmt.(parser.UnaryExpressionNode); ok {
		return e.emitInstruction(n.Expression)
	}
	if n, ok := stmt.(parser.BlockExpressionNode); ok {
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpOBlock, nil, nil))

		for _, stmt := range n.Statements {
			e.emitInstruction(stmt)
		}

		latest := e.insts[len(e.insts)-1]
		isEmpty := bytes.Compare(latest.GetLabel(), l) == 0
		l = e.generateLabel()
		if isEmpty {
			e.insts = append(e.insts, NewInstruction(l, OpSave, nil, nil))
		} else {
			e.insts = append(e.insts, NewInstruction(l, OpSave, latest.GetLabel(), nil))
		}

		e.insts = append(e.insts, NewInstruction(e.generateLabel(), OpCBlock, nil, nil))
		return l
	}
	if n, ok := stmt.(parser.BooleanExpression); ok {
		ll := e.emitInstruction(n.Left)
		lr := e.emitInstruction(n.Right)
		var op byte
		switch fmt.Sprintf("%s", n.Operation.Token.GetMatch()) {
		case "equals":
			op = OpEquals
		case "different":
			op = OpDiff
		case "bigger":
			op = OpBigger
		case "smaller":
			op = OpSmaller
		}
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, op, ll, lr))
		return l
	}
	if n, ok := stmt.(parser.BinaryExpressionNode); ok {
		ll := e.emitInstruction(n.Left)
		lr := e.emitInstruction(n.Right)
		var op byte
		switch fmt.Sprintf("%s", n.Operation.Token.GetMatch()) {
		case "*":
			op = OpMultiply
		case "+":
			op = OpAdd
		case "-":
			op = OpSubstract
		case "/":
			op = OpDivide
		case "^":
			op = OpExponential
		}
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, op, ll, lr))
		return l
	}
	if n, ok := stmt.(parser.NumberLiteralNode); ok {
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpSave, byteutil.FromUint64(n.Value), nil))
		return l
	}
	if n, ok := stmt.(parser.IdLiteralNode); ok {
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpLoad, n.Token.GetMatch(), nil))
		return l
	}
	if n, ok := stmt.(parser.CalleeLiteralNode); ok {
		for _, p := range n.Params {
			ll := e.emitInstruction(p.Expression)
			l := e.generateLabel()
			e.insts = append(e.insts, NewInstruction(l, OpSaveParam, ll, nil))
		}
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpCall, n.Id.Token.GetMatch(), nil))

		l = e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpResult, nil, nil))
		return l
	}
	if n, ok := stmt.(parser.CallPrintStatementNode); ok {
		ll := e.emitInstruction(n.Param)
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpPrint, ll, nil))
		return l
	}
	return make([]byte, 8)
}

func (e *emt) Emit() ([]Instruction, error) {
	for _, stmt := range e.ast.Module.Statements {
		e.emitInstruction(stmt)
	}
	return e.insts, nil
}

func New(ast parser.AST) *emt {
	return &emt{ast, 0, make([]Instruction, 0)}
}
