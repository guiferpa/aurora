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
		{name: "a reference", operand: RefTo([]byte("07")), kind: Ref, bytes: []byte("07")},
		{name: "an immediate", operand: ImmOf([]byte{1, 2}), kind: Imm, bytes: []byte{1, 2}},
		{name: "a number", operand: ImmNum(3), kind: Imm, bytes: []byte{0, 0, 0, 0, 0, 0, 0, 3}},
		{name: "a name", operand: NameOf("x"), kind: Name, bytes: []byte("x")},
		{name: "a target", operand: TargetAt(3), kind: Target, bytes: []byte{0, 0, 0, 0, 0, 0, 0, 3}},
		{name: "some text", operand: TextOf("hi"), kind: Text, bytes: []byte("hi")},
		{name: "nothing", operand: Nothing(), kind: Empty, bytes: []byte{}},
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
	for _, operand := range []Operand{Nothing(), RefTo(nil), ImmOf(nil)} {
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
		{operand: ImmNum(1), want: "imm 0x0000000000000001"},
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
