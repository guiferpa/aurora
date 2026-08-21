package emitter

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/ir"
)

type Emitter interface {
	Emit(tree ast.AST) ([]ir.Instruction, error)
	EmitProgram(tree ast.AST) (ir.Program, error)
}

type emt struct {
	tapeSize int
}

func GenerateLabel(tc *int) []byte {
	t := []byte(fmt.Sprintf("0%d", *tc))
	*tc++
	return t
}

// printOpCodes maps each reading of a value to the instruction that writes it.
var printOpCodes = map[ast.PrintFormat]byte{
	ast.PrintBytes:   ir.OpPrintBytes,
	ast.PrintChars:   ir.OpPrintChars,
	ast.PrintDecimal: ir.OpPrintDecimal,
}

func EmitInstruction(tc *int, insts *[]ir.Instruction, expr ast.Node, tapeSize int) ir.Label {
	switch n := expr.(type) {
	case ast.IdentLiteral:
		return emitIdentLiteral(tc, insts, n, tapeSize)
	case ast.BlockExpression:
		return emitBlockExpression(tc, insts, n, tapeSize)
	case ast.DeferExpression:
		return emitDeferExpression(tc, insts, n, tapeSize)
	case ast.UnaryExpression:
		return emitUnaryExpression(tc, insts, n, tapeSize)
	case ast.RelativeExpression:
		return emitRelativeExpression(tc, insts, n, tapeSize)
	case ast.BooleanExpression:
		return emitBooleanExpression(tc, insts, n, tapeSize)
	case ast.TapeBracketExpression:
		return emitTapeBracketExpression(tc, insts, n, tapeSize)
	case ast.ShapeDeclaration:
		return emitShapeDeclaration(tc, insts, n, tapeSize)
	case ast.UseDeclaration:
		return emitUseDeclaration(tc, insts, n, tapeSize)
	case ast.ShapeLiteral:
		return emitShapeLiteral(tc, insts, n, tapeSize)
	case ast.FieldExpression:
		return emitFieldExpression(tc, insts, n, tapeSize)
	case ast.ShapedExpression:
		return emitShapedExpression(tc, insts, n, tapeSize)
	case ast.PullExpression:
		return emitPullExpression(tc, insts, n, tapeSize)
	case ast.HeadExpression:
		return emitHeadExpression(tc, insts, n, tapeSize)
	case ast.TailExpression:
		return emitTailExpression(tc, insts, n, tapeSize)
	case ast.PushExpression:
		return emitPushExpression(tc, insts, n, tapeSize)
	case ast.IfExpression:
		return emitIfExpression(tc, insts, n, tapeSize)
	case ast.CalleeLiteral:
		return emitCalleeLiteral(tc, insts, n, tapeSize)
	case ast.PrintStatement:
		return emitPrintStatement(tc, insts, n, tapeSize)
	case ast.AssertStatement:
		return emitAssertStatement(tc, insts, n, tapeSize)
	case ast.FeedExpression:
		return emitFeedExpression(tc, insts, n, tapeSize)
	case ast.BinaryExpression:
		return emitBinaryExpression(tc, insts, n, tapeSize)
	case ast.NumberLiteral:
		return emitNumberLiteral(tc, insts, n, tapeSize)
	case ast.TextLiteral:
		return emitTextLiteral(tc, insts, n, tapeSize)
	case ast.BooleanLiteral:
		return emitBooleanLiteral(tc, insts, n, tapeSize)
	case ast.IdentifierLiteral:
		return emitIdentifierLiteral(tc, insts, n, tapeSize)
	}

	// A node this does not know emits nothing, and answers with the neutral value.
	return byteutil.FalseTape(tapeSize)
}

// emitIdentLiteral binds a name to a value.
func emitIdentLiteral(tc *int, insts *[]ir.Instruction, n ast.IdentLiteral, tapeSize int) ir.Label {
	ll := []byte(n.Id)
	lr := EmitInstruction(tc, insts, n.Value, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpIdent, ll, lr))
	// A binding has a value like every other expression — the neutral one — and the
	// label has to come back, or a scope ending in a binding returns the fallback of
	// a node the emitter did not recognise.
	return l

}

// emitBlockExpression opens a scope and returns the value its body ended with.
func emitBlockExpression(tc *int, insts *[]ir.Instruction, n ast.BlockExpression, tapeSize int) ir.Label {
	var l []byte
	body := make([]ir.Instruction, 0)
	for _, ins := range n.Body {
		l = EmitInstruction(tc, &body, ins, tapeSize)
	}

	lsc := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(lsc, ir.OpBeginScope, nil, nil))

	body = append(body, ir.NewInstruction(GenerateLabel(tc), ir.OpReturn, lsc, l))
	*insts = append(*insts, body...)

	return lsc

}

// emitDeferExpression stores a scope to be run later, and skips over its body.
func emitDeferExpression(tc *int, insts *[]ir.Instruction, n ast.DeferExpression, tapeSize int) ir.Label {
	body := make([]ir.Instruction, 0)
	l := EmitInstruction(tc, &body, n.Block, tapeSize)
	bodylength := uint64(len(body))
	lo := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(lo, ir.OpDefer, l, byteutil.FromUint64(bodylength)))
	*insts = append(*insts, body...)
	return lo

}

// emitUnaryExpression negates: a tape is unsigned, so this is zero minus the value.
func emitUnaryExpression(tc *int, insts *[]ir.Instruction, n ast.UnaryExpression, tapeSize int) ir.Label {
	// A tape is unsigned, so negating is taking the value away from zero and letting it
	// wrap: -5 is the same tape as 0 - 5. Emitting the subtraction says exactly that,
	// and every stage below — the evaluator, the lowering, the EVM writer — already
	// knows how to subtract.
	//
	// The operator used to be dropped here and only the operand emitted, so `-5` was
	// 5 and `10 + -5` was 15.
	lo := EmitInstruction(tc, insts, n.Expression, tapeSize)

	lz := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(lz, ir.OpSave, byteutil.FalseTape(tapeSize), nil))

	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSubtract, lz, lo))
	return l

}

// emitRelativeExpression compares two values.
func emitRelativeExpression(tc *int, insts *[]ir.Instruction, n ast.RelativeExpression, tapeSize int) ir.Label {
	ll := EmitInstruction(tc, insts, n.Left, tapeSize)
	lr := EmitInstruction(tc, insts, n.Right, tapeSize)
	var op byte
	switch string(n.Operation.Token.GetMatch()) {
	case "equals":
		op = ir.OpEquals
	case "different":
		op = ir.OpDiff
	case "bigger":
		op = ir.OpBigger
	case "smaller":
		op = ir.OpSmaller
	}
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, op, ll, lr))

	return l

}

// emitBooleanExpression combines two truth values.
func emitBooleanExpression(tc *int, insts *[]ir.Instruction, n ast.BooleanExpression, tapeSize int) ir.Label {
	ll := EmitInstruction(tc, insts, n.Left, tapeSize)
	lr := EmitInstruction(tc, insts, n.Right, tapeSize)
	var op byte
	switch string(n.Operation.Token.GetMatch()) {
	case "or":
		op = ir.OpOr
	case "and":
		op = ir.OpAnd
	}
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, op, ll, lr))
	return l

}

// emitTapeBracketExpression builds a tape byte by byte.
func emitTapeBracketExpression(tc *int, insts *[]ir.Instruction, n ast.TapeBracketExpression, tapeSize int) ir.Label {
	// Create the initial tape, all zeros
	l := GenerateLabel(tc)
	tape := byteutil.FalseTape(tapeSize)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, tape, nil))

	// For each item, generate instruction and use ir.OpPull to add bytes directly
	for _, i := range n.Items {
		la := GenerateLabel(tc)
		li := EmitInstruction(tc, insts, i, tapeSize)
		*insts = append(*insts, ir.NewInstruction(la, ir.OpPull, l, li))
		l = la
	}
	return l

}

// emitShapeDeclaration answers the neutral value: a declaration does no work.
func emitShapeDeclaration(tc *int, insts *[]ir.Instruction, _ ast.ShapeDeclaration, tapeSize int) ir.Label {
	// A declaration emits no work. It still answers with a value, because everything in
	// Aurora is an expression, and the neutral one is what a declaration is worth.
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, byteutil.FalseTape(tapeSize), nil))
	return l

}

// emitUseDeclaration answers the neutral value: an import is resolved before the emitter runs.
func emitUseDeclaration(tc *int, insts *[]ir.Instruction, _ ast.UseDeclaration, tapeSize int) ir.Label {
	// Like a shape declaration, and for the same reason: it declares a name the compiler
	// uses and the program never touches, so there is no work to do and still a value to
	// answer with.
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, byteutil.FalseTape(tapeSize), nil))
	return l

}

// emitShapeLiteral lays one tape per field, end to end.
func emitShapeLiteral(tc *int, insts *[]ir.Instruction, n ast.ShapeLiteral, tapeSize int) ir.Label {
	// One tape per field, laid end to end — the shape a reel of the same length has.
	// Instructions carry two operands, so the run is built by chaining, the way a tape
	// literal chains ir.OpPull.
	//
	// It starts from an empty run rather than from the first field so that every field
	// crosses the same join, which is what narrows each one to a single tape. Starting
	// at the first field left that one whole, and a reel there became several tapes.
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, make([]byte, 0), nil))

	for _, value := range n.Values {
		lv := EmitInstruction(tc, insts, value, tapeSize)
		lj := GenerateLabel(tc)
		*insts = append(*insts, ir.NewInstruction(lj, ir.OpJoin, l, lv))
		l = lj
	}
	return l

}

// emitFieldExpression reads one tape out of a run, by an index resolved while parsing.
func emitFieldExpression(tc *int, insts *[]ir.Instruction, n ast.FieldExpression, tapeSize int) ir.Label {
	// The index was resolved while parsing, so it goes in as an immediate — the same
	// shape head and tail use. Nothing here knows the field had a name.
	lv := EmitInstruction(tc, insts, n.Expression, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpField, lv, byteutil.FromUint64(n.Index)))
	return l

}

// emitShapedExpression emits what was shaped: `as` is a claim, not an operation.
func emitShapedExpression(tc *int, insts *[]ir.Instruction, n ast.ShapedExpression, tapeSize int) ir.Label {
	// `as` says how to read a value, which is a question the compiler answers and the
	// program never asks: what is left is the value itself.
	return EmitInstruction(tc, insts, n.Expression, tapeSize)

}

// emitPullExpression shifts a tape left, the value entering at the right.
func emitPullExpression(tc *int, insts *[]ir.Instruction, n ast.PullExpression, tapeSize int) ir.Label {
	lt := EmitInstruction(tc, insts, n.Target, tapeSize)
	li := EmitInstruction(tc, insts, n.Item, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpPull, lt, li))
	return l

}

// emitHeadExpression keeps the first bytes of a tape.
func emitHeadExpression(tc *int, insts *[]ir.Instruction, n ast.HeadExpression, tapeSize int) ir.Label {
	e := EmitInstruction(tc, insts, n.Expression, tapeSize)
	ln := byteutil.FromUint64(n.Length)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpHead, e, ln))
	return l

}

// emitTailExpression drops the first bytes of a tape.
func emitTailExpression(tc *int, insts *[]ir.Instruction, n ast.TailExpression, tapeSize int) ir.Label {
	e := EmitInstruction(tc, insts, n.Expression, tapeSize)
	ln := byteutil.FromUint64(n.Length)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpTail, e, ln))
	return l

}

// emitPushExpression shifts a tape right, the value entering at the left.
func emitPushExpression(tc *int, insts *[]ir.Instruction, n ast.PushExpression, tapeSize int) ir.Label {
	lt := EmitInstruction(tc, insts, n.Target, tapeSize)
	li := EmitInstruction(tc, insts, n.Item, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpPush, lt, li))
	return l

}

// emitIfExpression lays out the test, the body and the else, with the jumps between them.
func emitIfExpression(tc *int, insts *[]ir.Instruction, n ast.IfExpression, tapeSize int) ir.Label {
	var bl, eul []byte

	/*Extract Else body*/
	euze := make([]ir.Instruction, 0)
	if n.Else != nil {
		for _, inst := range n.Else.Body {
			eul = EmitInstruction(tc, &euze, inst, tapeSize)
		}
	}
	euzelen := byteutil.FromUint64(uint64(len(euze)) + 1)

	/*Extract Condition body*/
	body := make([]ir.Instruction, 0)
	for _, inst := range n.Body {
		bl = EmitInstruction(tc, &body, inst, tapeSize)
	}
	bodylen := byteutil.FromUint64(uint64(len(body)) + 2)

	lt := EmitInstruction(tc, insts, n.Test, tapeSize)
	inl := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(inl, ir.OpIf, lt, bodylen))

	body = append(body, ir.NewInstruction(GenerateLabel(tc), ir.OpReturn, inl, bl))
	body = append(body, ir.NewInstruction(GenerateLabel(tc), ir.OpJump, euzelen, nil))
	*insts = append(*insts, body...)

	euze = append(euze, ir.NewInstruction(GenerateLabel(tc), ir.OpReturn, inl, eul))
	*insts = append(*insts, euze...)

	return inl

}

// emitCalleeLiteral applies values to a scope.
func emitCalleeLiteral(tc *int, insts *[]ir.Instruction, n ast.CalleeLiteral, tapeSize int) ir.Label {
	for i, p := range n.Params {
		ll := EmitInstruction(tc, insts, p.Expression, tapeSize)
		l := GenerateLabel(tc)
		*insts = append(*insts, ir.NewInstruction(l, ir.OpPushFeed, byteutil.FromUint64(uint64(i)), ll))
	}
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpCall, []byte(n.Id.Value), nil))
	return l

}

// emitPrintStatement writes a value in one of the three readings.
func emitPrintStatement(tc *int, insts *[]ir.Instruction, n ast.PrintStatement, tapeSize int) ir.Label {
	ll := EmitInstruction(tc, insts, n.Param, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, printOpCodes[n.Format], ll, nil))
	return l

}

// emitAssertStatement checks a condition, carrying its message as bytes.
func emitAssertStatement(tc *int, insts *[]ir.Instruction, n ast.AssertStatement, tapeSize int) ir.Label {
	// The message rides in the instruction as the bytes it is, not as a value: it is
	// written for whoever reads the result, and a value would have to fit in a tape.
	cond := EmitInstruction(tc, insts, n.Condition, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpAssert, cond, []byte(n.Message)))
	return l

}

// emitFeedExpression reads the nth value applied to this scope.
func emitFeedExpression(tc *int, insts *[]ir.Instruction, n ast.FeedExpression, tapeSize int) ir.Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpGetFeed, byteutil.FromUint64(n.Nth.Value), nil))
	return l

}

// emitBinaryExpression does arithmetic on two values.
func emitBinaryExpression(tc *int, insts *[]ir.Instruction, n ast.BinaryExpression, tapeSize int) ir.Label {
	ll := EmitInstruction(tc, insts, n.Left, tapeSize)
	lr := EmitInstruction(tc, insts, n.Right, tapeSize)
	var op byte
	switch string(n.Operation.Token.GetMatch()) {
	case "*":
		op = ir.OpMultiply
	case "+":
		op = ir.OpAdd
	case "-":
		op = ir.OpSubtract
	case "/":
		op = ir.OpDivide
	case "^":
		op = ir.OpExponential
	}

	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, op, ll, lr))

	return l

}

// emitNumberLiteral saves a number as a tape.
func emitNumberLiteral(tc *int, insts *[]ir.Instruction, n ast.NumberLiteral, tapeSize int) ir.Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, byteutil.PaddingTape(byteutil.FromUint64(n.Value), tapeSize), nil))
	return l

}

// emitTextLiteral saves text as the tape of its bytes.
func emitTextLiteral(tc *int, insts *[]ir.Instruction, n ast.TextLiteral, tapeSize int) ir.Label {
	// Text is a tape holding its bytes, so it saves like any other value.
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, n.Value, nil))
	return l

}

// emitBooleanLiteral saves true or false, which are tapes like any other.
func emitBooleanLiteral(tc *int, insts *[]ir.Instruction, n ast.BooleanLiteral, tapeSize int) ir.Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, n.Value, nil))
	return l

}

// emitIdentifierLiteral loads what a name is bound to.
func emitIdentifierLiteral(tc *int, insts *[]ir.Instruction, n ast.IdentifierLiteral, tapeSize int) ir.Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpLoad, []byte(n.Value), nil))
	return l

}

func (e *emt) Emit(tree ast.AST) ([]ir.Instruction, error) {
	program, err := e.EmitProgram(tree)
	if err != nil {
		return nil, err
	}
	return program.Instructions, nil
}

// EmitProgram compiles ast and records where each top-level expression landed, so a caller
// can run them one at a time and report each value as it is produced.
func (e *emt) EmitProgram(tree ast.AST) (ir.Program, error) {
	tc := 0
	insts := make([]ir.Instruction, 0)
	exprs := make([]ir.Expression, 0, len(tree.Nodes))

	for _, node := range tree.Nodes {
		from := len(insts)
		label := EmitInstruction(&tc, &insts, node, e.tapeSize)
		exprs = append(exprs, ir.Expression{From: from, To: len(insts), Label: label})
	}

	warnings := checkDeferCapacity(tree.Nodes, e.tapeSize)
	warnings = append(warnings, checkAsserts(tree.Nodes)...)
	warnings = append(warnings, checkAppliedValues(tree.Nodes)...)

	return ir.Program{
		Instructions: insts,
		Expressions:  exprs,
		Warnings:     warnings,
	}, nil
}

type NewEmitterOptions struct {
	// TapeSize is the width in bytes of every value. Zero means the default (8).
	TapeSize int
}

func New(options NewEmitterOptions) *emt {
	return &emt{tapeSize: byteutil.TapeSize(options.TapeSize)}
}
