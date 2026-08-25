package emitter

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/ir"
	"github.com/guiferpa/aurora/wire/token"
)

type Emitter interface {
	Emit(tree ast.AST) ([]ir.Block, error)
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

// originOf answers where a token was written, so an instruction can point at it. A nil token
// returns nothing, which is what a phase after this one reads as "no place to point at".
func originOf(t token.Token) ir.Origin {
	if t == nil {
		return ir.Origin{}
	}
	return ir.Origin{Line: t.GetLine(), Column: t.GetColumn()}
}

// operandFor answers the operand a node is worth, emitting for it only when there is
// something to emit.
//
// A literal is a value the program wrote down: there is nothing to work out, so it goes into
// the instruction that uses it. Everything else is computed, so it is emitted and referenced.
//
// The saves this removes were a third of what the emitter produced, and nearly all of them
// were read exactly once — an instruction whose whole job was to give a name to a number that
// was already known.
func operandFor(tc *int, insts *[]ir.Instruction, node ast.Node, tapeSize int) ir.Operand {
	switch n := node.(type) {
	case ast.NumberLiteral:
		return ir.Imm(n.Value, tapeSize)
	case ast.TextLiteral:
		return ir.ImmOf(n.Value, tapeSize)
	case ast.BooleanLiteral:
		return ir.ImmOf(n.Value, tapeSize)
	}
	return ir.RefTo(EmitInstruction(tc, insts, node, tapeSize))
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

	// A node this does not know emits nothing, and returns the neutral value.
	return byteutil.FalseTape(tapeSize)
}

// emitIdentLiteral binds a name to a value.
func emitIdentLiteral(tc *int, insts *[]ir.Instruction, n ast.IdentLiteral, tapeSize int) ir.Label {
	lr := operandFor(tc, insts, n.Value, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpIdent, ir.NameOf(n.Id), lr).At(originOf(n.Token)))
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
	*insts = append(*insts, ir.NewInstruction(lsc, opBeginScope, ir.Nothing(), ir.Nothing()))

	body = append(body, ir.NewInstruction(GenerateLabel(tc), opReturn, ir.RefTo(lsc), ir.RefTo(l)))
	*insts = append(*insts, body...)

	return lsc

}

// emitDeferExpression stores a scope to be run later, and skips over its body.
func emitDeferExpression(tc *int, insts *[]ir.Instruction, n ast.DeferExpression, tapeSize int) ir.Label {
	body := make([]ir.Instruction, 0)
	l := EmitInstruction(tc, &body, n.Block, tapeSize)
	bodylength := uint64(len(body))
	lo := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(lo, opDefer, ir.RefTo(l), ir.TargetAt(bodylength)))
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
	lo := operandFor(tc, insts, n.Expression, tapeSize)

	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSubtract, ir.ImmOf(byteutil.FalseTape(tapeSize), tapeSize), lo).At(originOf(n.Operation.Token)))
	return l

}

// emitRelativeExpression compares two values.
func emitRelativeExpression(tc *int, insts *[]ir.Instruction, n ast.RelativeExpression, tapeSize int) ir.Label {
	ll := operandFor(tc, insts, n.Left, tapeSize)
	lr := operandFor(tc, insts, n.Right, tapeSize)
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
	*insts = append(*insts, ir.NewInstruction(l, op, ll, lr).At(originOf(n.Operation.Token)))

	return l

}

// emitBooleanExpression combines two truth values.
func emitBooleanExpression(tc *int, insts *[]ir.Instruction, n ast.BooleanExpression, tapeSize int) ir.Label {
	ll := operandFor(tc, insts, n.Left, tapeSize)
	lr := operandFor(tc, insts, n.Right, tapeSize)
	var op byte
	switch string(n.Operation.Token.GetMatch()) {
	case "or":
		op = ir.OpOr
	case "and":
		op = ir.OpAnd
	}
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, op, ll, lr).At(originOf(n.Operation.Token)))
	return l

}

// emitTapeBracketExpression builds a tape byte by byte.
func emitTapeBracketExpression(tc *int, insts *[]ir.Instruction, n ast.TapeBracketExpression, tapeSize int) ir.Label {
	// A tape starts as zeros and every item is pulled onto it, entering at the right.
	//
	// One instruction over as many items as there are, rather than a chain of two-operand
	// pulls: the items are one construction, and saying so is what saves whoever reads the
	// IR from recognising a chain as one thing.
	lz := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(lz, ir.OpSave, ir.ImmOf(byteutil.FalseTape(tapeSize), tapeSize), ir.Nothing()))

	operands := make([]ir.Operand, 0, len(n.Items)+1)
	operands = append(operands, ir.RefTo(lz))
	for _, item := range n.Items {
		operands = append(operands, operandFor(tc, insts, item, tapeSize))
	}

	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstructionOver(l, ir.OpPull, operands...).At(originOf(n.Token)))
	return l

}

// emitShapeDeclaration answers the neutral value: a declaration does no work.
func emitShapeDeclaration(tc *int, insts *[]ir.Instruction, _ ast.ShapeDeclaration, tapeSize int) ir.Label {
	// A declaration emits no work. It still returns a value, because everything in
	// Aurora is an expression, and the neutral one is what a declaration is worth.
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, ir.ImmOf(byteutil.FalseTape(tapeSize), tapeSize), ir.Nothing()))
	return l

}

// emitUseDeclaration answers the neutral value: an import is resolved before the emitter runs.
func emitUseDeclaration(tc *int, insts *[]ir.Instruction, _ ast.UseDeclaration, tapeSize int) ir.Label {
	// Like a shape declaration, and for the same reason: it declares a name the compiler
	// uses and the program never touches, so there is no work to do and still a value to
	// answer with.
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, ir.ImmOf(byteutil.FalseTape(tapeSize), tapeSize), ir.Nothing()))
	return l

}

// emitShapeLiteral lays one tape per field, end to end.
func emitShapeLiteral(tc *int, insts *[]ir.Instruction, n ast.ShapeLiteral, tapeSize int) ir.Label {
	// One tape per field, laid end to end — the shape a reel of the same length has. One
	// instruction over as many fields as there are: a shape is one construction, and it used
	// to be a chain of joins because an instruction could only hold two operands.
	//
	// Every field crosses the same narrowing, which is what makes each one a single tape: a
	// reel handed to a field would otherwise stay whole and the shape would come out longer
	// than it declared.
	//
	// The chain used to start from an empty run so that the first field crossed a join like
	// the rest. With one instruction they all cross the same one, so the empty run is gone —
	// which it had to be, since it was the one operand here that was not a tape.
	operands := make([]ir.Operand, 0, len(n.Values))
	for _, value := range n.Values {
		operands = append(operands, operandFor(tc, insts, value, tapeSize))
	}

	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstructionOver(l, ir.OpJoin, operands...).At(originOf(n.Token)))
	return l

}

// emitFieldExpression reads one tape out of a run, by an index resolved while parsing.
func emitFieldExpression(tc *int, insts *[]ir.Instruction, n ast.FieldExpression, tapeSize int) ir.Label {
	// The index was resolved while parsing, so it goes in as an immediate — the same
	// shape head and tail use. Nothing here knows the field had a name.
	//
	// How many tapes the run has goes in beside it, because where a field sits inside a run
	// cannot be worked out from the index alone. A run is tapes laid end to end and nothing
	// in it says where it ends, so reading the last one means counting from the end — and a
	// consumer keeping a run as a fixed-width value has no other way to find it. It is not
	// something the reader could work out either: the run may arrive under a name, as a value
	// applied to a scope, or as a field of another run, and in none of those is the
	// construction in sight. The declaration the index came from says both, so both are said.
	lv := operandFor(tc, insts, n.Expression, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstructionOver(
		l, ir.OpField, lv, ir.Const(n.Index, tapeSize), ir.Const(n.Fields, tapeSize)).At(originOf(n.Token)))
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
	lt := operandFor(tc, insts, n.Target, tapeSize)
	li := operandFor(tc, insts, n.Item, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpPull, lt, li).At(originOf(n.Token)))
	return l

}

// emitHeadExpression keeps the first bytes of a tape.
func emitHeadExpression(tc *int, insts *[]ir.Instruction, n ast.HeadExpression, tapeSize int) ir.Label {
	e := operandFor(tc, insts, n.Expression, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpHead, e, ir.Const(n.Length, tapeSize)).At(originOf(n.Token)))
	return l

}

// emitTailExpression drops the first bytes of a tape.
func emitTailExpression(tc *int, insts *[]ir.Instruction, n ast.TailExpression, tapeSize int) ir.Label {
	e := operandFor(tc, insts, n.Expression, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpTail, e, ir.Const(n.Length, tapeSize)).At(originOf(n.Token)))
	return l

}

// emitPushExpression shifts a tape right, the value entering at the left.
func emitPushExpression(tc *int, insts *[]ir.Instruction, n ast.PushExpression, tapeSize int) ir.Label {
	lt := operandFor(tc, insts, n.Target, tapeSize)
	li := operandFor(tc, insts, n.Item, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpPush, lt, li).At(originOf(n.Token)))
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
	} else {
		// An if with no else returns the neutral value on the path where the test
		// fails, and the arm says so rather than leaving whoever reads the IR to know it.
		// A consumer that has to put a value somewhere — a stack — has nothing to put
		// there otherwise, and one that does not would be reading an operand naming
		// nothing.
		eul = GenerateLabel(tc)
		euze = append(euze, ir.NewInstruction(eul, ir.OpSave, ir.ImmOf(byteutil.FalseTape(tapeSize), tapeSize), ir.Nothing()).At(originOf(n.Token)))
	}
	euzelen := ir.TargetAt(uint64(len(euze)) + 1)

	/*Extract Condition body*/
	body := make([]ir.Instruction, 0)
	for _, inst := range n.Body {
		bl = EmitInstruction(tc, &body, inst, tapeSize)
	}
	bodylen := ir.TargetAt(uint64(len(body)) + 2)

	lt := operandFor(tc, insts, n.Test, tapeSize)
	inl := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(inl, opIf, lt, bodylen).At(originOf(n.Token)))

	body = append(body, ir.NewInstruction(GenerateLabel(tc), opReturn, ir.RefTo(inl), ir.RefTo(bl)))
	body = append(body, ir.NewInstruction(GenerateLabel(tc), opJump, euzelen, ir.Nothing()).At(originOf(n.Token)))
	*insts = append(*insts, body...)

	euze = append(euze, ir.NewInstruction(GenerateLabel(tc), opReturn, ir.RefTo(inl), ir.RefTo(eul)))
	*insts = append(*insts, euze...)

	return inl

}

// emitCalleeLiteral applies values to a scope.
func emitCalleeLiteral(tc *int, insts *[]ir.Instruction, n ast.CalleeLiteral, tapeSize int) ir.Label {
	// The call carries the values applied to it, in the order they were written.
	//
	// They used to be written into the environ of whoever was calling, one instruction each,
	// and read back from there — which worked because both sides agreed on a place that the
	// IR never mentioned. Nothing said the call depended on those values, so a pass moving
	// instructions had nothing telling it not to, and a backend had no such place at all.
	//
	// Where the values sit while a call happens is a calling convention, and a calling
	// convention belongs to a target: an environ here, memory or a stack on a chain, locals
	// in WASM, registers in an ABI. The IR says what is passed, and each of them decides
	// where it goes.
	operands := make([]ir.Operand, 0, len(n.Params)+1)
	operands = append(operands, ir.NameOf(n.Id.Value))
	for _, param := range n.Params {
		operands = append(operands, operandFor(tc, insts, param.Expression, tapeSize))
	}

	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstructionOver(l, ir.OpCall, operands...).At(originOf(n.Id.Token)))
	return l

}

// emitPrintStatement writes a value in one of the three readings.
func emitPrintStatement(tc *int, insts *[]ir.Instruction, n ast.PrintStatement, tapeSize int) ir.Label {
	ll := operandFor(tc, insts, n.Param, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, printOpCodes[n.Format], ll, ir.Nothing()).At(originOf(n.Token)))
	return l

}

// emitAssertStatement checks a condition, carrying its message as bytes.
func emitAssertStatement(tc *int, insts *[]ir.Instruction, n ast.AssertStatement, tapeSize int) ir.Label {
	// The message rides in the instruction as the bytes it is, not as a value: it is
	// written for whoever reads the result, and a value would have to fit in a tape.
	cond := operandFor(tc, insts, n.Condition, tapeSize)
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpAssert, cond, ir.TextOf(n.Message)).At(originOf(n.Token)))
	return l

}

// emitFeedExpression reads the nth value applied to this scope.
func emitFeedExpression(tc *int, insts *[]ir.Instruction, n ast.FeedExpression, tapeSize int) ir.Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpGetFeed, ir.Const(n.Nth.Value, tapeSize), ir.Nothing()))
	return l

}

// emitBinaryExpression does arithmetic on two values.
func emitBinaryExpression(tc *int, insts *[]ir.Instruction, n ast.BinaryExpression, tapeSize int) ir.Label {
	ll := operandFor(tc, insts, n.Left, tapeSize)
	lr := operandFor(tc, insts, n.Right, tapeSize)
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
	*insts = append(*insts, ir.NewInstruction(l, op, ll, lr).At(originOf(n.Operation.Token)))

	return l

}

// emitNumberLiteral saves a number as a tape.
func emitNumberLiteral(tc *int, insts *[]ir.Instruction, n ast.NumberLiteral, tapeSize int) ir.Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, ir.Imm(n.Value, tapeSize), ir.Nothing()).At(originOf(n.Token)))
	return l

}

// emitTextLiteral saves text as the tape of its bytes.
func emitTextLiteral(tc *int, insts *[]ir.Instruction, n ast.TextLiteral, tapeSize int) ir.Label {
	// Text is a tape holding its bytes, so it saves like any other value.
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, ir.ImmOf(n.Value, tapeSize), ir.Nothing()).At(originOf(n.Token)))
	return l

}

// emitBooleanLiteral saves true or false, which are tapes like any other.
func emitBooleanLiteral(tc *int, insts *[]ir.Instruction, n ast.BooleanLiteral, tapeSize int) ir.Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpSave, ir.ImmOf(n.Value, tapeSize), ir.Nothing()).At(originOf(n.Token)))
	return l

}

// emitIdentifierLiteral loads what a name is bound to.
func emitIdentifierLiteral(tc *int, insts *[]ir.Instruction, n ast.IdentifierLiteral, tapeSize int) ir.Label {
	l := GenerateLabel(tc)
	*insts = append(*insts, ir.NewInstruction(l, ir.OpLoad, ir.NameOf(n.Value), ir.Nothing()).At(originOf(n.Token)))
	return l

}

func (e *emt) Emit(tree ast.AST) ([]ir.Block, error) {
	program, err := e.EmitProgram(tree)
	if err != nil {
		return nil, err
	}
	return program.Blocks, nil
}

// placed answers where each top-level expression begins among the blocks.
//
// It is the first of the expression's instructions that landed somewhere the program runs
// through. That last part is the whole of the care needed: an expression that writes a scope
// has instructions inside the scope's own blocks, and those are reached by being called rather
// than by the run arriving at them — so the place an expression begins is where its binding
// landed, not where its body did.
func placed(exprs []reach, insts []ir.Instruction, blocks []ir.Block, places map[string]ir.Point) []ir.Expression {
	runs := make(map[ir.BlockID]bool)
	for _, id := range ir.Reaches(blocks, 0) {
		runs[id] = true
	}

	out := make([]ir.Expression, 0, len(exprs))
	for _, expr := range exprs {
		at := ir.Expression{Label: expr.label}
		for over := expr.from; over < expr.to && over < len(insts); over++ {
			point, known := places[byteutil.ToHex(insts[over].GetLabel())]
			if known && runs[point.Block] {
				at.At = point
				break
			}
		}
		out = append(out, at)
	}
	return out
}

// A reach is one top-level expression while it is still a stretch of the instruction list the
// emitter builds on its way to the blocks. It does not leave this package: what a caller is
// given is where the expression begins among the blocks.
type reach struct {
	from, to int
	label    []byte
}

// EmitProgram compiles ast and records where each top-level expression landed, so a caller
// can run them one at a time and report each value as it is produced.
func (e *emt) EmitProgram(tree ast.AST) (ir.Program, error) {
	tc := 0
	insts := make([]ir.Instruction, 0)
	exprs := make([]reach, 0, len(tree.Nodes))

	for _, node := range tree.Nodes {
		from := len(insts)
		label := EmitInstruction(&tc, &insts, node, e.tapeSize)
		exprs = append(exprs, reach{from: from, to: len(insts), label: label})
	}

	warnings := checkDeferCapacity(tree.Nodes, e.tapeSize)
	warnings = append(warnings, checkAsserts(tree.Nodes)...)
	warnings = append(warnings, checkAppliedValues(tree.Nodes)...)

	blocks, places := placedBlocksOf(insts)

	return ir.Program{
		Blocks:      blocks,
		Expressions: placed(exprs, insts, blocks, places),
		Warnings:    warnings,
	}, nil
}

type NewEmitterOptions struct {
	// TapeSize is the width in bytes of every value. Zero means the default (8).
	TapeSize int
}

func New(options NewEmitterOptions) *emt {
	return &emt{tapeSize: byteutil.TapeSize(options.TapeSize)}
}
