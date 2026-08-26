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
