package ir

import "testing"

// The rule, in all sixteen pairs, because it is derived rather than listed and a derivation
// is exactly the kind of thing that is right about the cases somebody thought of.
func TestWhatMayCrossWhat(t *testing.T) {
	effects := []Effect{Pure, Reads, Writes, Escapes}
	names := map[Effect]string{Pure: "pure", Reads: "reads", Writes: "writes", Escapes: "escapes"}

	// A row per effect, in the order above: whether it may be put in the other order with
	// each of the four.
	want := map[Effect][]bool{
		//        pure  reads  writes  escapes
		Pure:    {true, true, true, true},
		Reads:   {true, true, false, false},
		Writes:  {true, false, false, false},
		Escapes: {true, false, false, false},
	}

	for _, a := range effects {
		for at, b := range effects {
			if got := MayCross(a, b); got != want[a][at] {
				t.Errorf("%s may cross %s = %v, want %v", names[a], names[b], got, want[a][at])
			}
		}
	}
}

// The rule reads the same from either side. It is about a pair, and a pair has no first.
func TestTheRuleDoesNotCareWhichWayItIsAsked(t *testing.T) {
	effects := []Effect{Pure, Reads, Writes, Escapes}
	for _, a := range effects {
		for _, b := range effects {
			if MayCross(a, b) != MayCross(b, a) {
				t.Errorf("asked as (%d, %d) it answers %v, and the other way %v", a, b, MayCross(a, b), MayCross(b, a))
			}
		}
	}
}

// What each opcode does, for the ones that do something. Everything else is Pure, which is
// the answer for an opcode nobody wrote down — including one that does not exist.
func TestWhatAnOpcodeDoesBesideLeavingAValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   byte
		want Effect
	}{
		{name: "a binding writes the frame", op: OpIdent, want: Writes},
		{name: "a load reads it", op: OpLoad, want: Reads},
		{name: "a literal touches nothing", op: OpSave, want: Pure},
		{name: "a print says something", op: OpPrintDecimal, want: Writes},
		{name: "arithmetic computes and nothing else", op: OpAdd, want: Pure},
		{name: "a call is a jump inside one contract, so far", op: OpCall, want: Pure},
		{name: "an opcode nobody wrote down", op: 0xFF, want: Pure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EffectOf(tc.op); got != tc.want {
				t.Errorf("effect = %d, want %d", got, tc.want)
			}
		})
	}
}

// What outlives the program that did it, which is a different question from the effect.
//
// Reading them as one is a mistake worth a test: by the effect, a binding writes, because the
// frame is memory and a load after a store must stay after it. But a frame is gone when the
// scope is, and nobody outside the program ever saw it. A print is sharper still — its effect
// is Writes and it reaches no bytecode at all.
//
// It is what tells a question from a change, so getting it wrong sends somebody to pay for an
// answer, or lets them ask for something they meant to keep.
func TestWhatOutlivesTheProgram(t *testing.T) {
	for _, tc := range []struct {
		name  string
		op    byte
		keeps bool
	}{
		{name: "keeping something under a key", op: OpStorageSet, keeps: true},
		{name: "reading what is kept", op: OpStorageGet, keeps: false},
		// Both of these are Writes, and neither is anything a chain keeps.
		{name: "binding a name", op: OpIdent, keeps: false},
		{name: "a log", op: OpPrintDecimal, keeps: false},
		{name: "arithmetic", op: OpAdd, keeps: false},
		{name: "an opcode nobody wrote down", op: 0xFF, keeps: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Keeps(tc.op); got != tc.keeps {
				t.Errorf("keeps = %v, want %v", got, tc.keeps)
			}
		})
	}
}

// And a scope keeps something when what it calls does, however far away.
func TestWhatAScopeKeepsFollowsWhatItCalls(t *testing.T) {
	blocks := []Block{
		{
			ID: 0,
			Insts: []Instruction{
				NewInstruction([]byte("01"), OpSave, BlockOf(2), Nothing()),
				NewInstruction([]byte("02"), OpIdent, NameOf("writer"), RefTo([]byte("01"))),
				NewInstruction([]byte("03"), OpSave, BlockOf(3), Nothing()),
				NewInstruction([]byte("04"), OpIdent, NameOf("binder"), RefTo([]byte("03"))),
			},
			Term: Ends(Nothing()),
		},
		{
			ID:    1,
			Insts: []Instruction{NewInstructionOver([]byte("01"), OpCall, NameOf("writer"))},
			Term:  Ends(RefTo([]byte("01"))),
		},
		{
			ID:    2,
			Insts: []Instruction{NewInstruction([]byte("01"), OpStorageSet, Imm(1, 8), Imm(2, 8))},
			Term:  Ends(RefTo([]byte("01"))),
		},
		{
			// Binds a name and prints, and keeps nothing.
			ID: 3,
			Insts: []Instruction{
				NewInstruction([]byte("01"), OpSave, Imm(7, 8), Nothing()),
				NewInstruction([]byte("02"), OpIdent, NameOf("x"), RefTo([]byte("01"))),
				NewInstruction([]byte("03"), OpPrintDecimal, RefTo([]byte("01")), Nothing()),
			},
			Term: Ends(RefTo([]byte("03"))),
		},
	}

	if !KeepsAnything(blocks, 1) {
		t.Error("a scope calling one that keeps something was said to keep nothing")
	}
	if !KeepsAnything(blocks, 2) {
		t.Error("a scope that keeps something was said to keep nothing")
	}
	if KeepsAnything(blocks, 3) {
		t.Error("a scope that binds a name and prints was said to keep something")
	}
}
