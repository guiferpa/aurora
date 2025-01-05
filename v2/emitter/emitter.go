package emitter

import (
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
	if n, ok := stmt.(parser.BlockExpressionNode); ok {
		cins := e.insts
		e.insts = make([]Instruction, 0)
		for _, ins := range n.Body {
			e.emitInstruction(ins)
		}
		var length uint64 = uint64(len(e.insts)) + 1 // Length of function
		e.insts = cins
		e.insts = append(e.insts, NewInstruction(e.generateLabel(), OpBeginScope, n.Ref, byteutil.FromUint64(length)))

		l := e.generateLabel()
		for _, ins := range n.Body {
			l = e.emitInstruction(ins)
		}

		rl := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(rl, OpReturn, l, nil))

		l = e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpSave, n.Ref, nil))

		return l
	}
	if n, ok := stmt.(parser.UnaryExpressionNode); ok {
		return e.emitInstruction(n.Expression)
	}
	/*
		if n, ok := stmt.(parser.BlockExpressionNode); ok {
			l := e.generateLabel()
			e.insts = append(e.insts, NewInstruction(l, OpBeginScope, nil, nil))

			for _, stmt := range n.Statements {
				e.emitInstruction(stmt)
			}

			latest := e.insts[len(e.insts)-1]
			isEmpty := bytes.Compare(latest.GetLabel(), l) == 0
			l = e.generateLabel()
			if isEmpty {
				e.insts = append(e.insts, NewInstruction(l, OpReturn, nil, nil))
			} else {
				e.insts = append(e.insts, NewInstruction(l, OpReturn, latest.GetLabel(), nil))
			}

			e.insts = append(e.insts, NewInstruction(e.generateLabel(), OpEndScope, nil, nil))
			return l
		}
			if n, ok := stmt.(parser.ItemExpressionNode); ok {
				fmt.Print(n)
				return nil
			}
			if n, ok := stmt.(parser.BranchExpressionNode); ok {
				for _, it := range n.Items {
					l := e.generateLabel()
					fmt.Println(l, it)
					// TODO: This op must be a sequence of if expressions with else, currenlty is missing ELSE token, create ELSE support then keep the development

					// e.insts = append(e.insts, NewInstruction(l, Op))
				}
				return nil
			}
			if n, ok := stmt.(parser.RelativeExpression); ok {
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
			if n, ok := stmt.(parser.BooleanExpression); ok {
				ll := e.emitInstruction(n.Left)
				lr := e.emitInstruction(n.Right)
				var op byte
				switch fmt.Sprintf("%s", n.Operation.Token.GetMatch()) {
				case "or":
					op = OpOr
				case "and":
					op = OpAnd
				}
				l := e.generateLabel()
				e.insts = append(e.insts, NewInstruction(l, op, ll, lr))
				return l
			}
			if n, ok := stmt.(parser.IfExpressionNode); ok {
				cins := e.insts
				e.insts = make([]Instruction, 0)
				for _, ins := range n.Body {
					e.emitInstruction(ins)
				}
				var length uint64 = uint64(len(e.insts)) + 1 // Length of if body

				e.insts = cins
				lt := e.emitInstruction(n.Test)
				l := e.generateLabel()
				if n.Else != nil {
					e.insts = append(e.insts, NewInstruction(l, OpIfNot, lt, byteutil.FromUint64(length+1)))
				} else {
					e.insts = append(e.insts, NewInstruction(l, OpIfNot, lt, byteutil.FromUint64(length)))
				}

				for _, ins := range n.Body {
					e.emitInstruction(ins)
				}

				latest := e.insts[len(e.insts)-1]
				isEmpty := bytes.Compare(latest.GetLabel(), l) == 0
				l = e.generateLabel()
				if isEmpty {
					e.insts = append(e.insts, NewInstruction(l, OpSave, nil, nil))
				} else {
					e.insts = append(e.insts, NewInstruction(l, OpSave, latest.GetLabel(), nil))
				}

				if n.Else != nil {
					cins = e.insts
					for _, ins := range n.Else.Body {
						e.emitInstruction(ins)
					}

					var euzelength uint64 = uint64(len(e.insts)) + 1 // Length of else body
					e.insts = cins
					e.insts = append(e.insts, NewInstruction(e.generateLabel(), OpJump, byteutil.FromUint64(length+euzelength), nil))

					for _, ins := range n.Else.Body {
						e.emitInstruction(ins)
					}
				}

				return l
			}
	*/
	if n, ok := stmt.(parser.CalleeLiteralNode); ok {
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpPreCall, n.Id.Token.GetMatch(), nil))

		for _, p := range n.Params {
			ll := e.emitInstruction(p.Expression)
			l := e.generateLabel()
			e.insts = append(e.insts, NewInstruction(l, OpPushArg, ll, nil))
		}

		l = e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpCall, n.Id.Token.GetMatch(), nil))

		l = e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpResult, nil, nil))

		// rrl := e.generateLabel()
		// e.insts = append(e.insts, NewInstruction(rrl, OpLoad, l, nil))
		return l
	}
	if n, ok := stmt.(parser.PrintStatementNode); ok {
		ll := e.emitInstruction(n.Param)
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpPrint, ll, nil))
		return l
	}
	if n, ok := stmt.(parser.ArgumentsExpressionNode); ok {
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpGetArg, byteutil.FromUint64(n.Nth.Value), nil))
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
	if n, ok := stmt.(parser.BooleanLiteralNode); ok {
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpSave, n.Value, nil))
		return l
	}
	if n, ok := stmt.(parser.IdLiteralNode); ok {
		l := e.generateLabel()
		e.insts = append(e.insts, NewInstruction(l, OpLoad, n.Token.GetMatch(), nil))
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
