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

	if problems := Verify(blocks); len(problems) != 4 {
		t.Errorf("it found %d things wrong, want the four that are: %v", len(problems), problems)
	}
}
