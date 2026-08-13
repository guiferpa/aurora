package evm

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
)

// The push opcodes are contiguous from PUSH1 to PUSH32, so the tape width picks the opcode
// and the operand carries exactly that many bytes.
func TestWritePushUsesTheTapeWidth(t *testing.T) {
	cases := []struct {
		name    string
		operand []byte
		size    int
		want    []byte
	}{
		{name: "one byte", operand: []byte{7}, size: 1, want: []byte{OpPush1, 7}},
		{name: "two bytes", operand: []byte{1, 44}, size: 2, want: []byte{OpPush2, 1, 44}},
		{name: "eight bytes", operand: []byte{1}, size: 8, want: []byte{OpPush8, 0, 0, 0, 0, 0, 0, 0, 1}},
		{name: "narrower operand is padded", operand: []byte{9}, size: 4, want: []byte{OpPush4, 0, 0, 0, 9}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			if _, err := WritePush(buf, tc.operand, tc.size); err != nil {
				t.Fatalf("WritePush: %v", err)
			}
			if got := buf.Bytes(); !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWritePush32(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	if _, err := WritePush(buf, []byte{1}, 32); err != nil {
		t.Fatalf("WritePush: %v", err)
	}
	got := buf.Bytes()
	if got[0] != OpPush32 {
		t.Errorf("opcode = %#x, want PUSH32 (%#x)", got[0], OpPush32)
	}
	if len(got) != 33 {
		t.Errorf("wrote %d bytes, want 33 (opcode + 32)", len(got))
	}
}

// A boolean is a tape like any other value: it no longer gets pushed as a single byte.
func TestBooleanIsPushedAsATape(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	if _, err := WriteSave(buf, byteutil.TrueTape(byteutil.DefaultTapeSize), byteutil.DefaultTapeSize); err != nil {
		t.Fatalf("WriteSave: %v", err)
	}
	want := []byte{OpPush8, 0, 0, 0, 0, 0, 0, 0, 1}
	if got := buf.Bytes(); !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
