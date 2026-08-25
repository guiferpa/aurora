package ir

import (
	"strings"
	"testing"
)

// A scope is reached by being named and never by control arriving at it, so what a run passes
// through is the whole of what a terminator can lead to.
func TestReachesFollowsTerminatorsAndNothingElse(t *testing.T) {
	blocks := []Block{
		{ID: 0, Insts: []Instruction{NewInstruction([]byte("a"), 1, BlockOf(3), Nothing())}, Term: Goes(1)},
		{ID: 1, Term: Chooses(RefTo([]byte("c")), To(2), To(4))},
		{ID: 2, Term: Ends(Nothing())},
		{ID: 3, Term: Ends(Imm(1, 8))}, // a scope: named by block 0, never gone to
		{ID: 4, Term: Ends(Nothing())},
	}

	reached := Reaches(blocks, 0)
	seen := make(map[BlockID]bool, len(reached))
	for _, id := range reached {
		seen[id] = true
	}

	for _, id := range []BlockID{0, 1, 2, 4} {
		if !seen[id] {
			t.Errorf("block %d is gone to and was not reached", id)
		}
	}
	if seen[3] {
		t.Error("a scope was reached: a binding names a block, and naming is not going")
	}
	if len(reached) != 4 {
		t.Errorf("reached %d blocks, want the four the run passes through", len(reached))
	}
}

// A block reached twice is walked once. Two arms of a branch meet, so the block they meet at is
// named by both, and answering it twice would run it twice.
func TestReachesWalksABlockOnce(t *testing.T) {
	blocks := []Block{
		{ID: 0, Term: Chooses(RefTo([]byte("c")), To(1), To(2))},
		{ID: 1, Term: Goes(3, RefTo([]byte("x")))},
		{ID: 2, Term: Goes(3, RefTo([]byte("y")))},
		{ID: 3, Params: []Label{[]byte("m")}, Term: Ends(RefTo([]byte("m")))},
	}

	if got := len(Reaches(blocks, 0)); got != 4 {
		t.Errorf("reached %d blocks, want 4: the block the arms meet at is one block", got)
	}
}

// Everything that names a block moves with it. Getting this wrong is quiet — a block that
// exists, named by mistake — which is why it is one function and not a habit.
func TestShiftedMovesEverythingThatNamesABlock(t *testing.T) {
	blocks := []Block{
		{
			ID:    0,
			Insts: []Instruction{NewInstruction([]byte("a"), 1, BlockOf(1), Nothing()).At(Origin{Line: 3, Column: 4})},
			Term:  Chooses(RefTo([]byte("c")), To(1, RefTo([]byte("x"))), To(2)),
		},
		{ID: 1, Term: Goes(2)},
		{ID: 2, Term: Ends(Nothing())},
	}

	moved := Shifted(blocks, 10)

	if got := moved[0].ID; got != 10 {
		t.Errorf("the first block is %d, want 10", got)
	}
	if got := moved[0].Insts[0].GetLeft().Block(); got != 11 {
		t.Errorf("the binding names block %d, want 11", got)
	}
	if got := moved[0].Term.Targets[0].Block; got != 11 {
		t.Errorf("the first way out goes to block %d, want 11", got)
	}
	if got := moved[0].Term.Targets[1].Block; got != 12 {
		t.Errorf("the second way out goes to block %d, want 12", got)
	}

	// What is carried over stays carried over: the values handed to a block, and where an
	// instruction was written.
	if got := len(moved[0].Term.Targets[0].Args); got != 1 {
		t.Errorf("the way out hands over %d values, want the one it had", got)
	}
	if got := moved[0].Insts[0].GetOrigin(); got.Line != 3 || got.Column != 4 {
		t.Errorf("the instruction came from %v, want where it was written", got)
	}

	// And the blocks it came from are untouched, so joining a program does not spoil the
	// program it joined.
	if got := blocks[0].Term.Targets[0].Block; got != 1 {
		t.Errorf("shifting changed what it was given: block %d", got)
	}
}

// One file carries on into the next, and a scope is untouched by that — it is named rather than
// gone to, so it still ends by returning, which is what a call needs it to do.
func TestGoesOnToTurnsEndingsIntoGoings(t *testing.T) {
	blocks := []Block{
		{ID: 0, Insts: []Instruction{NewInstruction([]byte("a"), 1, BlockOf(2), Nothing())}, Term: Goes(1)},
		{ID: 1, Term: Ends(RefTo([]byte("v")))},
		{ID: 2, Term: Ends(Imm(9, 8))}, // the scope
	}

	joined := GoesOnTo(blocks, 0, 5)

	if got := joined[1].Term.Kind; got != Br {
		t.Errorf("the file ends with %v, want it carrying on", got)
	}
	if got := joined[1].Term.Targets[0].Block; got != 5 {
		t.Errorf("it carries on to block %d, want the next file", got)
	}
	if got := joined[2].Term.Kind; got != Ret {
		t.Errorf("the scope ends with %v, want it answering", got)
	}
}

// A program reads as blocks, because a list read in order was never the program in order: the
// block a run goes to next is the one its terminator names.
func TestFormatBlocksWritesEachWayOfEnding(t *testing.T) {
	blocks := []Block{
		{ID: 0, Insts: []Instruction{NewInstruction([]byte("a"), 1, Imm(2, 8), Nothing())},
			Term: Chooses(RefTo([]byte("a")), To(1), To(2))},
		{ID: 1, Term: Goes(2, RefTo([]byte("a")))},
		{ID: 2, Params: []Label{[]byte("m"), nil}, Term: Ends(RefTo([]byte("m")))},
	}

	got := FormatBlocks(blocks)

	for _, want := range []string{
		"b0()",
		"brif ref 0x61 -> b1(), b2()",
		"br b2(ref 0x61)",
		"b2(0x6D, _)", // a named parameter and one a scope takes positionally
		"ret ref 0x6D",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the program does not read as %q:\n%s", want, got)
		}
	}
}

// A scope is worth the number of its block, and that number is read off the operand rather
// than looked up.
func TestBlockOperandCarriesTheBlock(t *testing.T) {
	operand := BlockOf(258)

	if got := operand.Kind(); got != KindBlock {
		t.Errorf("kind is %v, want a block", got)
	}
	if got := operand.Block(); got != 258 {
		t.Errorf("it names block %d, want 258", got)
	}
	if got := operand.String(); !strings.HasPrefix(got, "block ") {
		t.Errorf("it reads as %q, want it saying what it is", got)
	}
}

// An origin is metadata: dropping it loses a diagnostic and never a meaning. Zero means the
// emitter had nothing to point at — a value it invented rather than one somebody wrote.
func TestOriginIsKnownOnlyWhenItPointsSomewhere(t *testing.T) {
	if (Origin{}).Known() {
		t.Error("an origin nobody set says it points somewhere")
	}
	if !(Origin{Line: 1, Column: 1}).Known() {
		t.Error("an origin at the first line says it points nowhere")
	}
}
