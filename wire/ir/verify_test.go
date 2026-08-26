package ir

import (
	"strings"
	"testing"
)

// Each of the things a program can be wrong about, and what it says about each.
//
// The point of the check is what it refuses, so this is the half that matters: a check that
// passes everything passes a broken program too, and nobody finds out until a contract answers
// something nobody asked for.
func TestWhatVerifyRefuses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		blocks []Block
		says   string
	}{
		{
			name: "a value read before anything leaves it",
			blocks: []Block{{
				ID:    0,
				Insts: []Instruction{NewInstruction([]byte("01"), OpAdd, RefTo([]byte("ff")), Imm(1, 8))},
				Term:  Ends(RefTo([]byte("01"))),
			}},
			says: "nothing before it leaves a value under that name",
		},
		{
			name: "a value read by the terminator that nothing leaves",
			blocks: []Block{{
				ID:   0,
				Term: Ends(RefTo([]byte("ff"))),
			}},
			says: "nothing in the block leaves a value under that name",
		},
		{
			name: "two values under one name",
			blocks: []Block{{
				ID: 0,
				Insts: []Instruction{
					NewInstruction([]byte("01"), OpSave, Imm(1, 8), Nothing()),
					NewInstruction([]byte("01"), OpSave, Imm(2, 8), Nothing()),
				},
				Term: Ends(RefTo([]byte("01"))),
			}},
			says: "and so does something before it",
		},
		{
			name:   "a block that goes where there is no block",
			blocks: []Block{{ID: 0, Term: Goes(7)}},
			says:   "it goes to block 7, and the program has 1",
		},
		{
			name: "a branch that hands over the wrong number of values",
			blocks: []Block{
				{ID: 0, Term: Goes(1)},
				{ID: 1, Params: []Label{[]byte("aa")}, Term: Ends(RefTo([]byte("aa")))},
			},
			says: "it hands 0 values to block 1, which takes 1",
		},
		{
			name: "a way of ending that names the wrong number of blocks",
			blocks: []Block{
				{ID: 0, Term: Terminator{Kind: BrIf, Cond: Imm(1, 8), Targets: []Target{{Block: 1}}}},
				{ID: 1, Term: Ends(Imm(1, 8))},
			},
			says: "chooses between two somewheres and names 1 blocks, and that way of ending names 2",
		},
		{
			name: "a value handed over that nothing leaves",
			blocks: []Block{
				{ID: 0, Term: Goes(1, RefTo([]byte("ff")))},
				{ID: 1, Params: []Label{[]byte("aa")}, Term: Ends(RefTo([]byte("aa")))},
			},
			// The label is named as every other reader of the IR names one, which is the hex of
			// its bytes: "ff" written here is 6666 there, and a message that spelled it
			// differently would be a name nobody could search for.
			says: "it hands over 6666, and nothing in the block leaves a value under that name",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			problems := Verify(tc.blocks)
			if len(problems) == 0 {
				t.Fatal("it verified a program that is wrong")
			}
			if !strings.Contains(problems[0].Error(), tc.says) {
				t.Errorf("it says %q, want it to say %q", problems[0], tc.says)
			}
			// Where, or nobody finds it.
			if !strings.HasPrefix(problems[0].Error(), "block ") {
				t.Errorf("it says %q, and never says where", problems[0])
			}
		})
	}
}

// A sound program has nothing said about it, which is the other half.
func TestVerifySaysNothingAboutASoundProgram(t *testing.T) {
	blocks := []Block{
		{
			ID: 0,
			Insts: []Instruction{
				NewInstruction([]byte("01"), OpSave, Imm(1, 8), Nothing()),
				NewInstruction([]byte("02"), OpAdd, RefTo([]byte("01")), Imm(2, 8)),
			},
			Term: Terminator{Kind: BrIf, Cond: RefTo([]byte("02")),
				Targets: []Target{{Block: 1, Args: []Operand{RefTo([]byte("02"))}}, {Block: 2, Args: []Operand{RefTo([]byte("01"))}}}},
		},
		{ID: 1, Params: []Label{[]byte("aa")}, Term: Ends(RefTo([]byte("aa")))},
		{ID: 2, Params: []Label{[]byte("bb")}, Term: Ends(RefTo([]byte("bb")))},
	}

	if problems := Verify(blocks); len(problems) != 0 {
		t.Errorf("it refused a sound program: %v", problems)
	}
}

// Everything wrong at once, and not the first of them.
//
// A program that is wrong is usually wrong in several places, and a check that stops at one
// turns fixing it into a loop of compile, read, fix, compile.
func TestVerifyAnswersEverythingWrongAtOnce(t *testing.T) {
	blocks := []Block{{
		ID: 0,
		Insts: []Instruction{
			NewInstruction([]byte("01"), OpAdd, RefTo([]byte("ff")), Nothing()),
			NewInstruction([]byte("01"), OpAdd, RefTo([]byte("ee")), Nothing()),
		},
		Term: Goes(9),
	}}

	// Six: each of the two adds reads a value nobody leaves and is missing its second
	// operand, the second leaves a value under a name the first already used, and the block
	// goes where there is no block.
	if problems := Verify(blocks); len(problems) != 6 {
		t.Errorf("it found %d things wrong, want the six that are: %v", len(problems), problems)
	}
}

// What an operand may be, which is the half of the check about meaning rather than shape.
//
// The same bytes are a number under one kind and a name under another, and which one an opcode
// wants used to be a comment beside it — a description kept in agreement by whoever remembered.
// It cost twice, and both times somebody found out afterwards.
func TestWhatAnOpcodeTakes(t *testing.T) {
	for _, tc := range []struct {
		name string
		inst Instruction
		says string
	}{
		{
			name: "a name where a value goes",
			inst: NewInstruction([]byte("01"), OpAdd, NameOf("x"), Imm(1, 8)),
			says: "operand 0 of opcode 2 is a name, and that operand takes a value",
		},
		{
			name: "a value where a name goes",
			inst: NewInstruction([]byte("01"), OpLoad, Imm(1, 8), Nothing()),
			says: "takes a name",
		},
		{
			name: "text where a value goes",
			inst: NewInstruction([]byte("01"), OpPrintDecimal, TextOf("hello"), Nothing()),
			says: "is text, and that operand takes a value",
		},
		{
			name: "something where nothing goes",
			inst: NewInstruction([]byte("01"), OpLoad, NameOf("x"), Imm(1, 8)),
			says: "takes nothing",
		},
		{
			name: "a name in the second half of a binding",
			inst: NewInstruction([]byte("01"), OpIdent, NameOf("x"), NameOf("y")),
			says: "operand 1 of opcode 6 is a name, and that operand takes a value",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			problems := Verify([]Block{{ID: 0, Insts: []Instruction{tc.inst}, Term: Ends(Nothing())}})
			if len(problems) == 0 {
				t.Fatal("it verified an instruction given something its opcode does not take")
			}
			said := false
			for _, problem := range problems {
				said = said || strings.Contains(problem.Error(), tc.says)
			}
			if !said {
				t.Errorf("it says %v, want one of them to say %q", problems, tc.says)
			}
		})
	}
}

// And the ones that are right are said nothing about — including the two an opcode's comment
// used to get wrong.
func TestWhatAnOpcodeTakesAndIsGiven(t *testing.T) {
	for _, tc := range []struct {
		name string
		inst Instruction
	}{
		{name: "a value written down where a value goes", inst: NewInstruction([]byte("01"), OpAdd, Imm(1, 8), Imm(2, 8))},
		{name: "a binding", inst: NewInstruction([]byte("01"), OpIdent, NameOf("x"), Imm(1, 8))},
		// A save carries a literal, and it carries the number of a block: a scope bound to a
		// name is the second, and the comment beside the opcode said Imm for a long time.
		{name: "a save of a literal", inst: NewInstruction([]byte("01"), OpSave, Imm(1, 8), Nothing())},
		{name: "a save of a scope", inst: NewInstruction([]byte("01"), OpSave, BlockOf(3), Nothing())},
		// A call takes a name and as many values as were applied, because a scope has no
		// arity — none of them is as right as any other.
		{name: "a call of no values", inst: NewInstructionOver([]byte("01"), OpCall, NameOf("f"))},
		{name: "a call of three", inst: NewInstructionOver([]byte("01"), OpCall, NameOf("f"), Imm(1, 8), Imm(2, 8), Imm(3, 8))},
		{name: "a run of five", inst: NewInstructionOver([]byte("01"), OpJoin, Imm(1, 8), Imm(2, 8), Imm(3, 8), Imm(4, 8), Imm(5, 8))},
		{name: "a field", inst: NewInstructionOver([]byte("01"), OpField, RefTo([]byte("00")), Const(1, 8), Const(3, 8))},
		{name: "an assertion", inst: NewInstruction([]byte("01"), OpAssert, RefTo([]byte("00")), TextOf("it holds"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blocks := []Block{{
				ID:     0,
				Params: []Label{[]byte("00")},
				Insts:  []Instruction{tc.inst},
				Term:   Ends(Nothing()),
			}}
			for _, problem := range Verify(blocks) {
				t.Errorf("%v", problem)
			}
		})
	}
}

// An opcode nobody wrote down is said nothing about, which is the honest answer for one that
// has not been described yet — and the reason this file is worth keeping in agreement.
func TestAnOpcodeNobodyWroteDownIsNotRefused(t *testing.T) {
	blocks := []Block{{
		ID:    0,
		Insts: []Instruction{NewInstruction([]byte("01"), 0xFE, NameOf("anything"), TextOf("at all"))},
		Term:  Ends(RefTo([]byte("01"))),
	}}

	if problems := Verify(blocks); len(problems) != 0 {
		t.Errorf("it refused an opcode nobody has described: %v", problems)
	}
}
