package emitter

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/parser"
)

type Emitter interface {
	Emit(ast parser.AST) ([]Instruction, error)
	EmitProgram(ast parser.AST) (Program, error)
}

// Expression is one top-level expression of a program: the range of instructions it
// compiled to, and the label the temp holding its value ends up under.
//
// A caller that wants to report values in source order needs this. Reading the temp map
// after the whole program has run cannot give it: the map has no order, and everything
// written by print along the way has already been emitted.
type Expression struct {
	From  int // index of its first instruction
	To    int // index one past its last instruction
	Label []byte
}

// Program is what a source file compiled to: the instruction stream, where each top-level
// expression sits inside it, and anything worth saying that did not stop the compilation.
type Program struct {
	Instructions []Instruction
	Expressions  []Expression
	Warnings     []Warning
}

type emt struct {
	tapeSize int
}

func GenerateLabel(tc *int) []byte {
	t := []byte(fmt.Sprintf("0%d", *tc))
	*tc++
	return t
}

type Label []byte

// printOpCodes maps each reading of a value to the instruction that writes it.
var printOpCodes = map[parser.PrintFormat]byte{
	parser.PrintBytes:   OpPrintBytes,
	parser.PrintChars:   OpPrintChars,
	parser.PrintDecimal: OpPrintDecimal,
}

func EmitInstruction(tc *int, insts *[]Instruction, expr parser.Node, tapeSize int) Label {
	switch n := expr.(type) {
	case parser.IdentLiteral:
		return emitIdentLiteral(tc, insts, n, tapeSize)
	case parser.BlockExpression:
		return emitBlockExpression(tc, insts, n, tapeSize)
	case parser.DeferExpression:
		return emitDeferExpression(tc, insts, n, tapeSize)
	case parser.UnaryExpression:
		return emitUnaryExpression(tc, insts, n, tapeSize)
	case parser.RelativeExpression:
		return emitRelativeExpression(tc, insts, n, tapeSize)
	case parser.BooleanExpression:
		return emitBooleanExpression(tc, insts, n, tapeSize)
	case parser.TapeBracketExpression:
		return emitTapeBracketExpression(tc, insts, n, tapeSize)
	case parser.StructDeclaration:
		return emitStructDeclaration(tc, insts, n, tapeSize)
	case parser.StructLiteral:
		return emitStructLiteral(tc, insts, n, tapeSize)
	case parser.FieldExpression:
		return emitFieldExpression(tc, insts, n, tapeSize)
	case parser.ShapedExpression:
		return emitShapedExpression(tc, insts, n, tapeSize)
	case parser.PullExpression:
		return emitPullExpression(tc, insts, n, tapeSize)
	case parser.HeadExpression:
		return emitHeadExpression(tc, insts, n, tapeSize)
	case parser.TailExpression:
		return emitTailExpression(tc, insts, n, tapeSize)
	case parser.PushExpression:
		return emitPushExpression(tc, insts, n, tapeSize)
	case parser.IfExpression:
		return emitIfExpression(tc, insts, n, tapeSize)
	case parser.CalleeLiteral:
		return emitCalleeLiteral(tc, insts, n, tapeSize)
	case parser.PrintStatement:
		return emitPrintStatement(tc, insts, n, tapeSize)
	case parser.AssertStatement:
		return emitAssertStatement(tc, insts, n, tapeSize)
	case parser.FeedExpression:
		return emitFeedExpression(tc, insts, n, tapeSize)
	case parser.BinaryExpression:
		return emitBinaryExpression(tc, insts, n, tapeSize)
	case parser.NumberLiteral:
		return emitNumberLiteral(tc, insts, n, tapeSize)
	case parser.TextLiteral:
		return emitTextLiteral(tc, insts, n, tapeSize)
	case parser.BooleanLiteral:
		return emitBooleanLiteral(tc, insts, n, tapeSize)
	case parser.IdentifierLiteral:
		return emitIdentifierLiteral(tc, insts, n, tapeSize)
	}

	// A node this does not know emits nothing, and answers with the neutral value.
	return byteutil.FalseTape(tapeSize)
}

// emitIdentLiteral binds a name to a value.
func emitIdentLiteral(tc *int, insts *[]Instruction, n parser.IdentLiteral, tapeSize int) Label {
	ll := n.Token.GetMatch()
	lr := EmitInstruction(tc, insts, n.Value, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpIdent, ll, lr))
	// A binding has a value like every other expression — the neutral one — and the
	// label has to come back, or a scope ending in a binding returns the fallback of
	// a node the emitter did not recognise.
	return l

}

// emitBlockExpression opens a scope and returns the value its body ended with.
func emitBlockExpression(tc *int, insts *[]Instruction, n parser.BlockExpression, tapeSize int) Label {
	var l []byte
	body := make([]Instruction, 0)
	for _, ins := range n.Body {
		l = EmitInstruction(tc, &body, ins, tapeSize)
	}

	lsc := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(lsc, OpBeginScope, nil, nil))

	body = append(body, NewInstruction(GenerateLabel(tc), OpReturn, lsc, l))
	*insts = append(*insts, body...)

	return lsc

}

// emitDeferExpression stores a scope to be run later, and skips over its body.
func emitDeferExpression(tc *int, insts *[]Instruction, n parser.DeferExpression, tapeSize int) Label {
	body := make([]Instruction, 0)
	l := EmitInstruction(tc, &body, n.Block, tapeSize)
	bodylength := uint64(len(body))
	lo := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(lo, OpDefer, l, byteutil.FromUint64(bodylength)))
	*insts = append(*insts, body...)
	return lo

}

// emitUnaryExpression negates: a tape is unsigned, so this is zero minus the value.
func emitUnaryExpression(tc *int, insts *[]Instruction, n parser.UnaryExpression, tapeSize int) Label {
	// A tape is unsigned, so negating is taking the value away from zero and letting it
	// wrap: -5 is the same tape as 0 - 5. Emitting the subtraction says exactly that,
	// and every stage below — the evaluator, the lowering, the EVM writer — already
	// knows how to subtract.
	//
	// The operator used to be dropped here and only the operand emitted, so `-5` was
	// 5 and `10 + -5` was 15.
	lo := EmitInstruction(tc, insts, n.Expression, tapeSize)

	lz := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(lz, OpSave, byteutil.FalseTape(tapeSize), nil))

	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpSubtract, lz, lo))
	return l

}

// emitRelativeExpression compares two values.
func emitRelativeExpression(tc *int, insts *[]Instruction, n parser.RelativeExpression, tapeSize int) Label {
	ll := EmitInstruction(tc, insts, n.Left, tapeSize)
	lr := EmitInstruction(tc, insts, n.Right, tapeSize)
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

// emitBooleanExpression combines two truth values.
func emitBooleanExpression(tc *int, insts *[]Instruction, n parser.BooleanExpression, tapeSize int) Label {
	ll := EmitInstruction(tc, insts, n.Left, tapeSize)
	lr := EmitInstruction(tc, insts, n.Right, tapeSize)
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

// emitTapeBracketExpression builds a tape byte by byte.
func emitTapeBracketExpression(tc *int, insts *[]Instruction, n parser.TapeBracketExpression, tapeSize int) Label {
	// Create the initial tape, all zeros
	l := GenerateLabel(tc)
	tape := byteutil.FalseTape(tapeSize)
	*insts = append(*insts, NewInstruction(l, OpSave, tape, nil))

	// For each item, generate instruction and use OpPull to add bytes directly
	for _, i := range n.Items {
		la := GenerateLabel(tc)
		li := EmitInstruction(tc, insts, i, tapeSize)
		*insts = append(*insts, NewInstruction(la, OpPull, l, li))
		l = la
	}
	return l

}

// emitStructDeclaration answers the neutral value: a directive does no work.
func emitStructDeclaration(tc *int, insts *[]Instruction, _ parser.StructDeclaration, tapeSize int) Label {
	// A directive emits no work. It still answers with a value, because everything in
	// Aurora is an expression, and the neutral one is what a declaration is worth.
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpSave, byteutil.FalseTape(tapeSize), nil))
	return l

}

// emitStructLiteral lays one tape per field, end to end.
func emitStructLiteral(tc *int, insts *[]Instruction, n parser.StructLiteral, tapeSize int) Label {
	// One tape per field, laid end to end — the shape a reel of the same length has.
	// Instructions carry two operands, so the run is built by chaining, the way a tape
	// literal chains OpPull.
	//
	// It starts from an empty run rather than from the first field so that every field
	// crosses the same join, which is what narrows each one to a single tape. Starting
	// at the first field left that one whole, and a reel there became several tapes.
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpSave, make([]byte, 0), nil))

	for _, value := range n.Values {
		lv := EmitInstruction(tc, insts, value, tapeSize)
		lj := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(lj, OpJoin, l, lv))
		l = lj
	}
	return l

}

// emitFieldExpression reads one tape out of a run, by an index resolved while parsing.
func emitFieldExpression(tc *int, insts *[]Instruction, n parser.FieldExpression, tapeSize int) Label {
	// The index was resolved while parsing, so it goes in as an immediate — the same
	// shape head and tail use. Nothing here knows the field had a name.
	lv := EmitInstruction(tc, insts, n.Expression, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpField, lv, byteutil.FromUint64(n.Index)))
	return l

}

// emitShapedExpression emits what was shaped: `as` is a claim, not an operation.
func emitShapedExpression(tc *int, insts *[]Instruction, n parser.ShapedExpression, tapeSize int) Label {
	// `as` says how to read a value, which is a question the compiler answers and the
	// program never asks: what is left is the value itself.
	return EmitInstruction(tc, insts, n.Expression, tapeSize)

}

// emitPullExpression shifts a tape left, the value entering at the right.
func emitPullExpression(tc *int, insts *[]Instruction, n parser.PullExpression, tapeSize int) Label {
	lt := EmitInstruction(tc, insts, n.Target, tapeSize)
	li := EmitInstruction(tc, insts, n.Item, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpPull, lt, li))
	return l

}

// emitHeadExpression keeps the first bytes of a tape.
func emitHeadExpression(tc *int, insts *[]Instruction, n parser.HeadExpression, tapeSize int) Label {
	e := EmitInstruction(tc, insts, n.Expression, tapeSize)
	ln := byteutil.FromUint64(n.Length)
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpHead, e, ln))
	return l

}

// emitTailExpression drops the first bytes of a tape.
func emitTailExpression(tc *int, insts *[]Instruction, n parser.TailExpression, tapeSize int) Label {
	e := EmitInstruction(tc, insts, n.Expression, tapeSize)
	ln := byteutil.FromUint64(n.Length)
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpTail, e, ln))
	return l

}

// emitPushExpression shifts a tape right, the value entering at the left.
func emitPushExpression(tc *int, insts *[]Instruction, n parser.PushExpression, tapeSize int) Label {
	lt := EmitInstruction(tc, insts, n.Target, tapeSize)
	li := EmitInstruction(tc, insts, n.Item, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpPush, lt, li))
	return l

}

// emitIfExpression lays out the test, the body and the else, with the jumps between them.
func emitIfExpression(tc *int, insts *[]Instruction, n parser.IfExpression, tapeSize int) Label {
	var bl, eul []byte

	/*Extract Else body*/
	euze := make([]Instruction, 0)
	if n.Else != nil {
		for _, inst := range n.Else.Body {
			eul = EmitInstruction(tc, &euze, inst, tapeSize)
		}
	}
	euzelen := byteutil.FromUint64(uint64(len(euze)) + 1)

	/*Extract Condition body*/
	body := make([]Instruction, 0)
	for _, inst := range n.Body {
		bl = EmitInstruction(tc, &body, inst, tapeSize)
	}
	bodylen := byteutil.FromUint64(uint64(len(body)) + 2)

	lt := EmitInstruction(tc, insts, n.Test, tapeSize)
	inl := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(inl, OpIf, lt, bodylen))

	body = append(body, NewInstruction(GenerateLabel(tc), OpReturn, inl, bl))
	body = append(body, NewInstruction(GenerateLabel(tc), OpJump, euzelen, nil))
	*insts = append(*insts, body...)

	euze = append(euze, NewInstruction(GenerateLabel(tc), OpReturn, inl, eul))
	*insts = append(*insts, euze...)

	return inl

}

// emitCalleeLiteral applies values to a scope.
func emitCalleeLiteral(tc *int, insts *[]Instruction, n parser.CalleeLiteral, tapeSize int) Label {
	for i, p := range n.Params {
		ll := EmitInstruction(tc, insts, p.Expression, tapeSize)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpPushFeed, byteutil.FromUint64(uint64(i)), ll))
	}
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpCall, n.Id.Token.GetMatch(), nil))
	return l

}

// emitPrintStatement writes a value in one of the three readings.
func emitPrintStatement(tc *int, insts *[]Instruction, n parser.PrintStatement, tapeSize int) Label {
	ll := EmitInstruction(tc, insts, n.Param, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, printOpCodes[n.Format], ll, nil))
	return l

}

// emitAssertStatement checks a condition, carrying its message as bytes.
func emitAssertStatement(tc *int, insts *[]Instruction, n parser.AssertStatement, tapeSize int) Label {
	// The message rides in the instruction as the bytes it is, not as a value: it is
	// written for whoever reads the result, and a value would have to fit in a tape.
	cond := EmitInstruction(tc, insts, n.Condition, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpAssert, cond, []byte(n.Message)))
	return l

}

// emitFeedExpression reads the nth value applied to this scope.
func emitFeedExpression(tc *int, insts *[]Instruction, n parser.FeedExpression, tapeSize int) Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpGetFeed, byteutil.FromUint64(n.Nth.Value), nil))
	return l

}

// emitBinaryExpression does arithmetic on two values.
func emitBinaryExpression(tc *int, insts *[]Instruction, n parser.BinaryExpression, tapeSize int) Label {
	ll := EmitInstruction(tc, insts, n.Left, tapeSize)
	lr := EmitInstruction(tc, insts, n.Right, tapeSize)
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

// emitNumberLiteral saves a number as a tape.
func emitNumberLiteral(tc *int, insts *[]Instruction, n parser.NumberLiteral, tapeSize int) Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpSave, byteutil.PaddingTape(byteutil.FromUint64(n.Value), tapeSize), nil))
	return l

}

// emitTextLiteral saves text as the tape of its bytes.
func emitTextLiteral(tc *int, insts *[]Instruction, n parser.TextLiteral, tapeSize int) Label {
	// Text is a tape holding its bytes, so it saves like any other value.
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpSave, n.Value, nil))
	return l

}

// emitBooleanLiteral saves true or false, which are tapes like any other.
func emitBooleanLiteral(tc *int, insts *[]Instruction, n parser.BooleanLiteral, tapeSize int) Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpSave, n.Value, nil))
	return l

}

// emitIdentifierLiteral loads what a name is bound to.
func emitIdentifierLiteral(tc *int, insts *[]Instruction, n parser.IdentifierLiteral, tapeSize int) Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, NewInstruction(l, OpLoad, n.Token.GetMatch(), nil))
	return l

}

func (e *emt) Emit(ast parser.AST) ([]Instruction, error) {
	program, err := e.EmitProgram(ast)
	if err != nil {
		return nil, err
	}
	return program.Instructions, nil
}

// EmitProgram compiles ast and records where each top-level expression landed, so a caller
// can run them one at a time and report each value as it is produced.
func (e *emt) EmitProgram(ast parser.AST) (Program, error) {
	tc := 0
	insts := make([]Instruction, 0)
	exprs := make([]Expression, 0, len(ast.Nodes))

	for _, node := range ast.Nodes {
		from := len(insts)
		label := EmitInstruction(&tc, &insts, node, e.tapeSize)
		exprs = append(exprs, Expression{From: from, To: len(insts), Label: label})
	}

	return Program{
		Instructions: insts,
		Expressions:  exprs,
		Warnings:     append(checkDeferCapacity(ast.Nodes, e.tapeSize), checkAsserts(ast.Nodes)...),
	}, nil
}

type NewEmitterOptions struct {
	// TapeSize is the width in bytes of every value. Zero means the default (8).
	TapeSize int
}

func New(options NewEmitterOptions) *emt {
	return &emt{tapeSize: byteutil.TapeSize(options.TapeSize)}
}
