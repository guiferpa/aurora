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
	enableLogging bool
	tapeSize      int
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
	if n, ok := expr.(parser.IdentLiteral); ok {
		ll := n.Token.GetMatch()
		lr := EmitInstruction(tc, insts, n.Value, tapeSize)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpIdent, ll, lr))
		// A binding has a value like every other expression — the neutral one — and the
		// label has to come back, or a scope ending in a binding returns the fallback of
		// a node the emitter did not recognise.
		return l
	}
	if n, ok := expr.(parser.BlockExpression); ok {
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
	if n, ok := expr.(parser.DeferExpression); ok {
		body := make([]Instruction, 0)
		l := EmitInstruction(tc, &body, n.Block, tapeSize)
		bodylength := uint64(len(body))
		lo := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(lo, OpDefer, l, byteutil.FromUint64(bodylength)))
		*insts = append(*insts, body...)
		return lo
	}
	if n, ok := expr.(parser.UnaryExpression); ok {
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
	if n, ok := expr.(parser.RelativeExpression); ok {
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
	if n, ok := expr.(parser.BooleanExpression); ok {
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
	if n, ok := expr.(parser.TapeBracketExpression); ok {
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
	if _, ok := expr.(parser.StructDeclaration); ok {
		// A directive emits no work. It still answers with a value, because everything in
		// Aurora is an expression, and the neutral one is what a declaration is worth.
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpSave, byteutil.FalseTape(tapeSize), nil))
		return l
	}
	if n, ok := expr.(parser.StructLiteral); ok {
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
	if n, ok := expr.(parser.FieldExpression); ok {
		// The index was resolved while parsing, so it goes in as an immediate — the same
		// shape head and tail use. Nothing here knows the field had a name.
		lv := EmitInstruction(tc, insts, n.Expression, tapeSize)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpField, lv, byteutil.FromUint64(n.Index)))
		return l
	}
	if n, ok := expr.(parser.ShapedExpression); ok {
		// `as` says how to read a value, which is a question the compiler answers and the
		// program never asks: what is left is the value itself.
		return EmitInstruction(tc, insts, n.Expression, tapeSize)
	}
	if n, ok := expr.(parser.PullExpression); ok {
		lt := EmitInstruction(tc, insts, n.Target, tapeSize)
		li := EmitInstruction(tc, insts, n.Item, tapeSize)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpPull, lt, li))
		return l
	}
	if n, ok := expr.(parser.HeadExpression); ok {
		e := EmitInstruction(tc, insts, n.Expression, tapeSize)
		ln := byteutil.FromUint64(n.Length)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpHead, e, ln))
		return l
	}
	if n, ok := expr.(parser.TailExpression); ok {
		e := EmitInstruction(tc, insts, n.Expression, tapeSize)
		ln := byteutil.FromUint64(n.Length)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpTail, e, ln))
		return l
	}
	if n, ok := expr.(parser.PushExpression); ok {
		lt := EmitInstruction(tc, insts, n.Target, tapeSize)
		li := EmitInstruction(tc, insts, n.Item, tapeSize)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpPush, lt, li))
		return l
	}
	if n, ok := expr.(parser.IfExpression); ok {
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
	if n, ok := expr.(parser.CalleeLiteral); ok {
		for i, p := range n.Params {
			ll := EmitInstruction(tc, insts, p.Expression, tapeSize)
			l := GenerateLabel(tc)
			*insts = append(*insts, NewInstruction(l, OpPushFeed, byteutil.FromUint64(uint64(i)), ll))
		}
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpCall, n.Id.Token.GetMatch(), nil))
		return l
	}
	if n, ok := expr.(parser.PrintStatement); ok {
		ll := EmitInstruction(tc, insts, n.Param, tapeSize)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, printOpCodes[n.Format], ll, nil))
		return l
	}
	if n, ok := expr.(parser.AssertStatement); ok {
		// The message rides in the instruction as the bytes it is, not as a value: it is
		// written for whoever reads the result, and a value would have to fit in a tape.
		cond := EmitInstruction(tc, insts, n.Condition, tapeSize)
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpAssert, cond, []byte(n.Message)))
		return l
	}
	if n, ok := expr.(parser.FeedExpression); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpGetFeed, byteutil.FromUint64(n.Nth.Value), nil))
		return l
	}
	if n, ok := expr.(parser.BinaryExpression); ok {
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
	if n, ok := expr.(parser.NumberLiteral); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpSave, byteutil.PaddingTape(byteutil.FromUint64(n.Value), tapeSize), nil))
		return l
	}
	if n, ok := expr.(parser.TextLiteral); ok {
		// Text is a tape holding its bytes, so it saves like any other value.
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpSave, n.Value, nil))
		return l
	}
	if n, ok := expr.(parser.BooleanLiteral); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpSave, n.Value, nil))
		return l
	}
	if n, ok := expr.(parser.IdentifierLiteral); ok {
		l := GenerateLabel(tc)
		*insts = append(*insts, NewInstruction(l, OpLoad, n.Token.GetMatch(), nil))
		return l
	}
	return byteutil.FalseTape(tapeSize)
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
	EnableLogging bool
	// TapeSize is the width in bytes of every value. Zero means the default (8).
	TapeSize int
}

func New(options NewEmitterOptions) *emt {
	return &emt{enableLogging: options.EnableLogging, tapeSize: byteutil.TapeSize(options.TapeSize)}
}
