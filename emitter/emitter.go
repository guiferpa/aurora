package emitter

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/parser"
)

type Emitter interface {
	Emit(ast parser.AST) ([]Instruction, error)
}

type emt struct {
	enableLogging bool
}

func GenerateLabel(tc *int) []byte {
	t := []byte(fmt.Sprintf("0%d", *tc))
	*tc++
	return t
}

type Label []byte

func EmitInstruction(tc *int, insts *[]Instruction, stmt parser.Node) Label {
	if n, ok := stmt.(parser.Statement); ok {
		return EmitInstruction(tc, insts, n.Node)
	}
	if n, ok := stmt.(parser.IdentStatement); ok {
		ll := n.Token.GetMatch()
		lr := EmitInstruction(tc, insts, n.Expression)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpIdent, ll, lr))
	}
	if n, ok := stmt.(parser.BlockExpression); ok {
		var l []byte
		body := make([]Instruction, 0)
		for _, ins := range n.Body {
			l = EmitInstruction(tc, &body, ins)
		}

		lsc := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(lsc, OpBeginScope, nil, nil))

		body = append(body, NewInstruction(GenerateLabel(tc), OpReturn, lsc, l))
		*insts = append(*insts, body...)

		return lsc
	}
	if n, ok := stmt.(parser.DeferExpression); ok {
		body := make([]Instruction, 0)
		l := EmitInstruction(tc, &body, n.Block)
		bodylength := uint64(len(body))
		lo := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(lo, OpDefer, l, byteutil.FromUint64(bodylength)))
		*insts = append(*insts, body...)
		return lo
	}
	if n, ok := stmt.(parser.UnaryExpression); ok {
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

		return l
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
	if n, ok := stmt.(parser.TapeBracketExpression); ok {
		// Create initial tape with 8 bytes (all zeros)
		l := GenerateLabel(tc)
		tape := make([]byte, 8)
		*insts = append(*insts, NewInstruction(l, OpSave, tape, nil))

		// For each item, generate instruction and use OpPull to add bytes directly
		for _, i := range n.Items {
			la := GenerateLabel(tc)
			li := EmitInstruction(tc, insts, i)
			*insts = append(*insts, NewInstruction(la, OpPull, l, li))
			l = la
		}
		return l
	}
	if n, ok := stmt.(parser.PullExpression); ok {
		lt := EmitInstruction(tc, insts, n.Target)
		li := EmitInstruction(tc, insts, n.Item)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpPull, lt, li))
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
	if n, ok := stmt.(parser.IfExpression); ok {
		var bl, eul []byte

		/*Extract Else body*/
		euze := make([]Instruction, 0)
		if n.Else != nil {
			for _, inst := range n.Else.Body {
				eul = EmitInstruction(tc, &euze, inst)
			}
		}
		euzelen := byteutil.FromUint64(uint64(len(euze)) + 1)

		/*Extract Condition body*/
		body := make([]Instruction, 0)
		for _, inst := range n.Body {
			bl = EmitInstruction(tc, &body, inst)
		}
		bodylen := byteutil.FromUint64(uint64(len(body)) + 2)

		lt := EmitInstruction(tc, insts, n.Test)
		inl := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(inl, OpIf, lt, bodylen))

		body = append(body, NewInstruction(GenerateLabel(tc), OpReturn, inl, bl))
		body = append(body, NewInstruction(GenerateLabel(tc), OpJump, euzelen, nil))
		*insts = append(*insts, body...)

		euze = append(euze, NewInstruction(GenerateLabel(tc), OpReturn, inl, eul))
		*insts = append(*insts, euze...)

		return inl
	}
	if n, ok := stmt.(parser.CalleeLiteral); ok {
		for i, p := range n.Params {
			ll := EmitInstruction(tc, insts, p.Expression)
			l := GenerateLabel(tc)
			*insts = append(*insts, NewInstruction(l, OpPushArg, byteutil.FromUint64(uint64(i)), ll))
		}
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpCall, n.Id.Token.GetMatch(), nil))
		return l
	}
	if n, ok := stmt.(parser.PrintStatement); ok {
		ll := EmitInstruction(tc, insts, n.Param)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpPrint, ll, nil))
		return l
	}
	if n, ok := stmt.(parser.EchoStatement); ok {
		ll := EmitInstruction(tc, insts, n.Param)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpEcho, ll, nil))
		return l
	}
	if n, ok := stmt.(parser.AssertStatement); ok {
		cond := EmitInstruction(tc, insts, n.Condition)
		msg := EmitInstruction(tc, insts, n.Message)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpAssert, cond, msg))
		return l
	}
	if n, ok := stmt.(parser.ArgumentsExpression); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpGetArg, byteutil.FromUint64(n.Nth.Value), nil))
		return l
	}
	if n, ok := stmt.(parser.BinaryExpression); ok {
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

		return l
	}
	if n, ok := stmt.(parser.NumberLiteral); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpSave, byteutil.FromUint64(n.Value), nil))
		return l
	}
	if n, ok := stmt.(parser.ReelLiteral); ok {
		// Reel is an array of tapes (each char is a tape of 8 bytes)
		// Store the complete reel by concatenating all tapes
		l := GenerateLabel(tc)
		// Concatenate all tapes into a single byte array
		reelBytes := make([]byte, 0, len(n.Value)*8)
		for _, tape := range n.Value {
			reelBytes = append(reelBytes, tape...)
		}
		*insts = append(*insts, NewInstruction(l, OpSave, reelBytes, nil))
		return l
	}
	if n, ok := stmt.(parser.BooleanLiteral); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpSave, n.Value, nil))
		return l
	}
	if n, ok := stmt.(parser.IdentifierLiteral); ok {
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

type NewEmitterOptions struct {
	EnableLogging bool
}

func New(options NewEmitterOptions) *emt {
	return &emt{enableLogging: options.EnableLogging}
}
