package ir

import (
	"strings"
	"testing"
)

// The IR is what the phases agree on, and until it moved here it had no tests of its own: the
// emitter and the builder each tested what they made of it, and nobody tested the thing
// itself. What follows is the contract — that an instruction never holds nothing where it
// promised bytes, that every opcode answers to a name, and that a program written down reads
// the same way every time.

// An instruction is handed to a builder and to an evaluator, and both index into its
// operands. Answering with nil where they expect bytes is how a move becomes a panic.
func TestNewInstructionNeverHoldsNil(t *testing.T) {
	cases := []struct {
		name  string
		left  []byte
		right []byte
	}{
		{name: "both given", left: []byte{1}, right: []byte{2}},
		{name: "no left", right: []byte{2}},
		{name: "no right", left: []byte{1}},
		{name: "neither"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inst := NewInstruction([]byte("00"), OpAdd, tc.left, tc.right)

			if inst.GetLeft() == nil {
				t.Error("the left operand came back as nothing")
			}
			if inst.GetRight() == nil {
				t.Error("the right operand came back as nothing")
			}
		})
	}
}

// What was put in is what comes out: the label, the opcode and the operands, unchanged.
func TestAnInstructionAnswersWithWhatItWasGiven(t *testing.T) {
	inst := NewInstruction([]byte("07"), OpMultiply, []byte{1, 2}, []byte{3})

	if got := string(inst.GetLabel()); got != "07" {
		t.Errorf("label is %q, want 07", got)
	}
	if got := inst.GetOpCode(); got != OpMultiply {
		t.Errorf("opcode is %d, want %d", got, OpMultiply)
	}
	if got := inst.GetLeft(); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("left is %v, want [1 2]", got)
	}
	if got := inst.GetRight(); len(got) != 1 || got[0] != 3 {
		t.Errorf("right is %v, want [3]", got)
	}
}

// A name is how the vocabulary is read — by a trace, by a test that failed, by whoever is
// looking for why an instruction is where it is. An opcode added without one shows up as
// "Unknown" in all three, which reads like a bug in the program rather than a gap here.
func TestEveryOpcodeAnswersToAName(t *testing.T) {
	for op := OpMultiply; op <= OpField; op++ {
		name := ResolveOpCode(op)

		if name == "Unknown" {
			t.Errorf("opcode %d has no name", op)
		}
		if !strings.HasPrefix(name, "Op") {
			t.Errorf("opcode %d is called %q, want a name starting with Op", op, name)
		}
	}
}

// Two opcodes answering to the same name would make a trace lie about which one ran.
func TestNoTwoOpcodesShareAName(t *testing.T) {
	seen := make(map[string]byte)

	for op := OpMultiply; op <= OpField; op++ {
		name := ResolveOpCode(op)
		if first, taken := seen[name]; taken {
			t.Errorf("%s names both %d and %d", name, first, op)
			continue
		}
		seen[name] = op
	}
}

// A byte that is not an opcode still has to answer something: a trace of a program being
// debugged is the worst place to panic.
func TestAByteThatIsNotAnOpcode(t *testing.T) {
	if got := ResolveOpCode(0xff); got != "Unknown" {
		t.Errorf("answered %q, want Unknown", got)
	}
}

// The written form is one instruction per line, in the order they run.
func TestFormat(t *testing.T) {
	insts := []Instruction{
		NewInstruction([]byte("00"), OpSave, []byte{1}, nil),
		NewInstruction([]byte("01"), OpAdd, []byte("00"), []byte("00")),
	}

	lines := strings.Split(strings.TrimRight(Format(insts), "\n"), "\n")

	if len(lines) != 2 {
		t.Fatalf("wrote %d lines for two instructions: %q", len(lines), lines)
	}
	if !strings.Contains(lines[0], "OpSave") {
		t.Errorf("the first line is %q, want it to name the opcode", lines[0])
	}
	if !strings.Contains(lines[1], "OpAdd") {
		t.Errorf("the second line is %q, want it to name the opcode", lines[1])
	}
}

// Nothing to write is nothing written, rather than a blank line: a caller that shows the
// trace of an empty program should show nothing at all.
func TestFormatOfNoInstructions(t *testing.T) {
	if got := Format(nil); got != "" {
		t.Errorf("wrote %q for no instructions", got)
	}
}
