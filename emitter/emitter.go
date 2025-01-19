package emitter

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/parser"
)

type Emitter interface {
	Emit(ast parser.AST) ([]Instruction, error)
}

type emt struct{}

func GenerateLabel(tc *int) []byte {
	t := []byte(fmt.Sprintf("0%d", *tc))
	*tc++
	return t
}

func EmitInstruction(tc *int, insts *[]Instruction, stmt parser.Node) []byte {
	if n, ok := stmt.(parser.StatementNode); ok {
		return EmitInstruction(tc, insts, n.Node)
	}
	if n, ok := stmt.(parser.IdentStatementNode); ok {
		ll := n.Token.GetMatch()
		lr := EmitInstruction(tc, insts, n.Expression)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpIdent, ll, lr))
	}
	if n, ok := stmt.(parser.BlockExpressionNode); ok {
		var l []byte
		body := make([]Instruction, 0)
		for _, ins := range n.Body {
			l = EmitInstruction(tc, &body, ins)
		}
		body = append(body, NewInstruction(GenerateLabel(tc), OpReturn, l, nil))

		var length uint64 = uint64(len(body)) // Length of function
		*insts = append(*insts, NewInstruction(GenerateLabel(tc), OpBeginScope, n.Ref, byteutil.FromUint64(length)))
		*insts = append(*insts, body...)

		l = GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpSave, n.Ref, nil))

		return l
	}
	if n, ok := stmt.(parser.UnaryExpressionNode); ok {
		return EmitInstruction(tc, insts, n.Expression)
	}
	if n, ok := stmt.(parser.RelativeExpression); ok {
		ll := EmitInstruction(tc, insts, n.Left)
		lr := EmitInstruction(tc, insts, n.Right)
		var op byte
		switch string(n.Operation.Token.GetMatch()) {
		case "equals":
			op = OpEquals
		case "different":
			op = OpDiff
		case "bigger":
			op = OpBigger
		case "smaller":
			op = OpSmaller
		}
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, op, ll, lr))

		rl := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(rl, OpResult, nil, nil))

		return rl
	}
	if n, ok := stmt.(parser.BooleanExpression); ok {
		ll := EmitInstruction(tc, insts, n.Left)
		lr := EmitInstruction(tc, insts, n.Right)
		var op byte
		switch string(n.Operation.Token.GetMatch()) {
		case "or":
			op = OpOr
		case "and":
			op = OpAnd
		}
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, op, ll, lr))
		return l
	}
	if n, ok := stmt.(parser.TapeExpression); ok {
		l := GenerateLabel(tc)
		tape := make([]byte, n.Length*8)
		*insts = append(*insts, NewInstruction(l, OpSave, tape, nil))
		return l
	}
	if n, ok := stmt.(parser.TapeBracketExpression); ok {
		ln := 2 // Minimum of length
		l := GenerateLabel(tc)
		if len(n.Items) > 0 {
			ln = len(n.Items)
		}
		tape := make([]byte, ln*8)
		*insts = append(*insts, NewInstruction(l, OpSave, tape, nil))
		for _, i := range n.Items {
			la := GenerateLabel(tc)
			li := EmitInstruction(tc, insts, i)
			*insts = append(*insts, NewInstruction(la, OpAppend, l, li))
			l = la
		}
		return l
	}
	if n, ok := stmt.(parser.AppendExpression); ok {
		lt := EmitInstruction(tc, insts, n.Target)
		li := EmitInstruction(tc, insts, n.Item)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpAppend, lt, li))
		return l
	}
	if n, ok := stmt.(parser.HeadExpression); ok {
		e := EmitInstruction(tc, insts, n.Expression)
		ln := byteutil.FromUint64(n.Length)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpHead, e, ln))
		return l
	}
	if n, ok := stmt.(parser.TailExpression); ok {
		e := EmitInstruction(tc, insts, n.Expression)
		ln := byteutil.FromUint64(n.Length)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpTail, e, ln))
		return l
	}
	if n, ok := stmt.(parser.PushExpression); ok {
		lt := EmitInstruction(tc, insts, n.Target)
		li := EmitInstruction(tc, insts, n.Item)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpPush, lt, li))
		return l
	}
	if n, ok := stmt.(parser.IfExpressionNode); ok {
		var l []byte

		/*Extract Else body*/
		euze := make([]Instruction, 0)
		if n.Else != nil {
			for _, inst := range n.Else.Body {
				l = EmitInstruction(tc, &euze, inst)
			}
		}
		euze = append(euze, NewInstruction(GenerateLabel(tc), OpReturn, l, nil))
		euzelen := byteutil.FromUint64(uint64(len(euze)))

		/*Extract Condition body*/
		body := make([]Instruction, 0)
		for _, inst := range n.Body {
			l = EmitInstruction(tc, &body, inst)
		}
		body = append(body, NewInstruction(GenerateLabel(tc), OpReturn, l, nil))
		body = append(body, NewInstruction(GenerateLabel(tc), OpJump, euzelen, nil))
		bodylen := byteutil.FromUint64(uint64(len(body)))

		lt := EmitInstruction(tc, insts, n.Test)
		inl := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(inl, OpIf, lt, bodylen))
		*insts = append(*insts, body...)
		*insts = append(*insts, euze...)

		rl := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(rl, OpResult, nil, nil))

		return rl
	}
	if n, ok := stmt.(parser.CalleeLiteralNode); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpPreCall, n.Id.Token.GetMatch(), nil))

		for _, p := range n.Params {
			ll := EmitInstruction(tc, insts, p.Expression)
			l := GenerateLabel(tc)
			*insts = append(*insts, NewInstruction(l, OpPushArg, ll, nil))
		}

		l = GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpCall, n.Id.Token.GetMatch(), nil))

		l = GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpResult, nil, nil))

		return l
	}
	if n, ok := stmt.(parser.PrintStatementNode); ok {
		ll := EmitInstruction(tc, insts, n.Param)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpPrint, ll, nil))
		return l
	}
	if n, ok := stmt.(parser.ArgumentsExpressionNode); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpGetArg, byteutil.FromUint64(n.Nth.Value), nil))
		return l
	}
	if n, ok := stmt.(parser.BinaryExpressionNode); ok {
		ll := EmitInstruction(tc, insts, n.Left)
		lr := EmitInstruction(tc, insts, n.Right)
		var op byte
		switch string(n.Operation.Token.GetMatch()) {
		case "*":
			op = OpMultiply
		case "+":
			op = OpAdd
		case "-":
			op = OpSubtract
		case "/":
			op = OpDivide
		case "^":
			op = OpExponential
		}

		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, op, ll, lr))

		rl := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(rl, OpResult, nil, nil))

		return rl
	}
	if n, ok := stmt.(parser.NumberLiteralNode); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpSave, byteutil.FromUint64(n.Value), nil))
		return l
	}
	if n, ok := stmt.(parser.BooleanLiteral); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpSave, n.Value, nil))
		return l
	}
	if n, ok := stmt.(parser.IdLiteralNode); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpLoad, n.Token.GetMatch(), nil))
		return l
	}
	return make([]byte, 8)
}

func (e *emt) Emit(ast parser.AST) ([]Instruction, error) {
	tc := 0
	insts := make([]Instruction, 0)
	for _, stmt := range ast.Module.Statements {
		EmitInstruction(&tc, &insts, stmt)
	}
	return insts, nil
}

func New() *emt {
	return &emt{}
}
