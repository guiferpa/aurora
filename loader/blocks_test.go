package loader

import (
	"testing"

	"github.com/guiferpa/aurora/wire/ir"
)

// Two files become one program. Each is compiled on its own and numbers its blocks from zero,
// so everything the second one names has to move — and the first stops answering and carries
// on into it, because a program made of several files is one run through all of them.
func TestJoiningTwoFilesMovesTheSecondAndChainsTheFirst(t *testing.T) {
	first := []ir.Block{
		{ID: 0, Insts: []ir.Instruction{ir.NewInstruction([]byte("a"), ir.OpIdent, ir.NameOf("f"), ir.BlockOf(1))}, Term: ir.Ends(ir.Nothing())},
		{ID: 1, Term: ir.Ends(ir.Imm(1, 8))},
	}
	second := []ir.Block{
		{ID: 0, Insts: []ir.Instruction{ir.NewInstruction([]byte("b"), ir.OpIdent, ir.NameOf("g"), ir.BlockOf(1))}, Term: ir.Ends(ir.Nothing())},
		{ID: 1, Term: ir.Ends(ir.Imm(2, 8))},
	}

	joined := ir.GoesOnTo(append([]ir.Block{}, first...), 0, 2)
	joined = append(joined, ir.Shifted(second, 2)...)

	if got := joined[0].Term.Kind; got != ir.Br {
		t.Errorf("the first file ends with %v, want it carrying on into the second", got)
	}
	if got := joined[0].Term.Targets[0].Block; got != 2 {
		t.Errorf("it carries on to block %d, want the second file's first", got)
	}

	// The scope of the first file still answers: a scope is named, not gone to, so joining
	// files does not reach it.
	if got := joined[1].Term.Kind; got != ir.Ret {
		t.Errorf("the first file's scope ends with %v, want it answering", got)
	}

	// Everything the second file names moved with it.
	if got := joined[2].Insts[0].GetRight().Block(); got != 3 {
		t.Errorf("the second file binds block %d, want the one its scope moved to", got)
	}
	if got := joined[3].ID; got != 3 {
		t.Errorf("the second file's scope is block %d, want 3", got)
	}
}

// A scope is reached by being named, never by control arriving at it, so joining files leaves
// every scope alone — including the last file's, which is the one nothing follows.
func TestReachesStopsAtScopes(t *testing.T) {
	blocks := []ir.Block{
		{ID: 0, Insts: []ir.Instruction{ir.NewInstruction([]byte("a"), ir.OpIdent, ir.NameOf("f"), ir.BlockOf(1))}, Term: ir.Goes(2)},
		{ID: 1, Term: ir.Ends(ir.Imm(1, 8))},
		{ID: 2, Term: ir.Ends(ir.Nothing())},
	}

	reached := ir.Reaches(blocks, 0)
	for _, id := range reached {
		if id == 1 {
			t.Error("the run reached a scope, which is named rather than gone to")
		}
	}
	if len(reached) != 2 {
		t.Errorf("the run is %d blocks, want the two it goes through", len(reached))
	}
}
