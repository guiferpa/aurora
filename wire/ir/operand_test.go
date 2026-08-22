package ir

import (
	"bytes"
	"testing"
)

// An operand carries bytes and says how they are meant to be read. The bytes alone never
// said: the same 0x03 is three, or the value labelled "03", or three instructions ahead.
func TestAnOperandSaysWhatItIs(t *testing.T) {
	cases := []struct {
		name    string
		operand Operand
		kind    Kind
		bytes   []byte
	}{
		{name: "a reference", operand: RefTo([]byte("07")), kind: KindRef, bytes: []byte("07")},
		{name: "a value written down", operand: ImmOf([]byte{1, 2}, 8), kind: KindImm, bytes: []byte{0, 0, 0, 0, 0, 0, 1, 2}},
		{name: "a number the program wrote", operand: Imm(3, 8), kind: KindImm, bytes: []byte{0, 0, 0, 0, 0, 0, 0, 3}},
		{name: "a number the operation takes", operand: Const(2, 8), kind: KindConst, bytes: []byte{0, 0, 0, 0, 0, 0, 0, 2}},
		{name: "a name", operand: NameOf("x"), kind: KindName, bytes: []byte("x")},
		{name: "a target", operand: TargetAt(3), kind: KindTarget, bytes: []byte{0, 0, 0, 0, 0, 0, 0, 3}},
		{name: "some text", operand: TextOf("hi"), kind: KindText, bytes: []byte("hi")},
		{name: "nothing", operand: Nothing(), kind: KindEmpty, bytes: []byte{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.operand.Kind(); got != tc.kind {
				t.Errorf("kind is %v, want %v", got, tc.kind)
			}
			if got := tc.operand.Bytes(); !bytes.Equal(got, tc.bytes) {
				t.Errorf("bytes are %v, want %v", got, tc.bytes)
			}
		})
	}
}

// An operand that is not there answers with an empty slice, never nil: that is what every
// reader of the IR has always been handed, and none of them checks.
func TestAnOperandThatIsNotThereStillAnswersWithBytes(t *testing.T) {
	for _, operand := range []Operand{Nothing(), RefTo(nil), ImmOf(nil, 8)} {
		if got := operand.Bytes(); got == nil {
			t.Errorf("%v answered with nil", operand)
		}
	}
}

// The IR is read by people, and half of an instruction used to be unreadable: two runs of
// hex with nothing saying which was a value and which was a place.
func TestAnOperandWritesItselfForAPerson(t *testing.T) {
	cases := []struct {
		operand Operand
		want    string
	}{
		{operand: RefTo([]byte{7}), want: "ref 0x07"},
		{operand: Imm(1, 8), want: "imm 0x0000000000000001"},
		{operand: Const(2, 8), want: "const 0x0000000000000002"},
		{operand: NameOf("x"), want: "name 0x78"},
		{operand: Nothing(), want: "-"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.operand.String(); got != tc.want {
				t.Errorf("wrote %q, want %q", got, tc.want)
			}
		})
	}
}

// An instruction over a run carries as many operands as it was given. A construction has as
// many parts as it has, and used to become a chain of two-operand instructions for whoever
// read the IR to recognise as one thing.
func TestAnInstructionCarriesAsManyOperandsAsItWasGiven(t *testing.T) {
	inst := NewInstructionOver([]byte("09"), OpJoin, RefTo([]byte("00")), RefTo([]byte("01")), RefTo([]byte("02")))

	if got := len(inst.GetOperands()); got != 3 {
		t.Fatalf("carries %d operands, want 3", got)
	}
	// The first two still answer as a pair, which is what an instruction taking one reads.
	if got := string(inst.GetLeft().Bytes()); got != "00" {
		t.Errorf("left is %q, want the first operand", got)
	}
	if got := string(inst.GetRight().Bytes()); got != "01" {
		t.Errorf("right is %q, want the second operand", got)
	}
}

// An operand that is not there answers as Nothing rather than panicking, because a reader
// asking for a pair should not have to know how many the instruction actually has.
func TestAnOperandBeyondWhatAnInstructionHasIsNothing(t *testing.T) {
	inst := NewInstructionOver([]byte("09"), OpSave, ImmOf([]byte{1}, 8))

	if got := inst.GetRight(); got.Kind() != KindEmpty {
		t.Errorf("the second operand of a one-operand instruction is %v, want nothing", got)
	}
}
