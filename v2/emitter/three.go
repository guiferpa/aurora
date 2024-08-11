package emitter

import (
	"fmt"

	"github.com/guiferpa/aurora/parser"
)

type emt struct {
	ast     parser.AST
	opcodes []string
}

func (e *emt) genTemp() string {
	return fmt.Sprintf("t%d", len(e.opcodes))
}

func (e *emt) emitNode(stmt parser.Node) string {
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
		op := fmt.Sprintf("%s", n.Operation.Token.GetMatch())
		t := e.genTemp()
		e.opcodes = append(e.opcodes, fmt.Sprintf("%s = %s %s %s", t, tl, op, tr))
		return t
	}
	if n, ok := stmt.(parser.NumberLiteralNode); ok {
		t := e.genTemp()
		e.opcodes = append(e.opcodes, fmt.Sprintf("%s = %d", t, n.Value))
		return t
	}
	return ""
}

func (e *emt) Emit() []string {
	for _, stmt := range e.ast.Root.Statements {
		e.emitNode(stmt.Statement)
	}
	return e.opcodes
}

func NewThree(ast parser.AST) *emt {
	return &emt{ast, make([]string, 0)}
}

// 10 * 2 + 8 ^ 2
// t0 = 10
// t1 = 2
// t2 = t0 * t1 -> 10 * 2 = 20
// t3 = 8
// t4 = 2
// t5 = t3 ^ t4 -> 8 ^ 2 = 64
// t6 = t2 + t5 -> 20 + 64 = 84
