package evm

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

// The constructor copies the runtime out of the code being deployed and hands it to the chain.
// The size goes in two bytes, and the offset it copies from is this block's own length —
// written once, in the constant, and read here rather than repeated.
func TestWriteInstantiateBlock(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteInstantiateBlock(bs, 8); err != nil {
		t.Fatalf("writing the instantiate block: %v", err)
	}

	expected := []byte{
		OpPush2, 0x00, 0x08,
		OpPush1, INSTANTIATE_BLOCK_SIZE,
		OpPush1, 0x00,
		OpCodeCopy,
		OpPush2, 0x00, 0x08,
		OpPush1, 0x00,
		OpReturn,
	}
	if got := bs.Bytes(); !bytes.Equal(got, expected) {
		t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
	if got := bs.Len(); got != INSTANTIATE_BLOCK_SIZE {
		t.Errorf("the block measures %d and says it measures %d — the offset it copies from is its own length", got, INSTANTIATE_BLOCK_SIZE)
	}
}

// A runtime past what one byte holds used to be truncated by the conversion, so the
// constructor asked for 96 bytes of a contract that had 352 and the chain kept the first 96.
// Three deferred scopes reached that.
func TestARuntimePastOneByteIsWrittenWhole(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteInstantiateBlock(bs, 352); err != nil {
		t.Fatalf("writing the instantiate block: %v", err)
	}

	got := bs.Bytes()
	if want := []byte{OpPush2, 0x01, 0x60}; !bytes.Equal(got[:3], want) {
		t.Errorf("it copies %v bytes, want %v — 352, not 352 modulo 256", byteutil.ToUpperHex(got[:3]), byteutil.ToUpperHex(want))
	}
}

func TestWriteAdd(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteAdd(bs); err != nil {
		t.Errorf("Error writing add: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{OpAdd}
	if !bytes.Equal(got, expected) {
		t.Errorf("Add: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

func TestWriteMultiply(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteMultiply(bs); err != nil {
		t.Errorf("Error writing multiply: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{OpMul}
	if !bytes.Equal(got, expected) {
		t.Errorf("Multiply: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

func TestWriteSubtract(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteSubtract(bs); err != nil {
		t.Errorf("Error writing subtract: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{OpSub}
	if !bytes.Equal(got, expected) {
		t.Errorf("Subtract: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

func TestWriteDivide(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteDivide(bs); err != nil {
		t.Errorf("Error writing divide: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{OpDiv}
	if !bytes.Equal(got, expected) {
		t.Errorf("Divide: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

func TestWriteSave(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	operand := []byte{1}
	if _, err := WriteSave(bs, operand, byteutil.DefaultTapeSize); err != nil {
		t.Errorf("Error writing save: %v", err)
		return
	}
	got := bs.Bytes()
	// Every value is a tape of the configured width: a one-byte operand is padded, not
	// pushed as PUSH1. There is no special case for booleans any more.
	expected := []byte{OpPush8, 0, 0, 0, 0, 0, 0, 0, 1}
	if !bytes.Equal(got, expected) {
		t.Errorf("Save: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

// A name is stored in a slot of memory of its own, and the address goes in two bytes. It used
// to go in one, and a slot is thirty-two wide, so the ninth name was given the address of the
// first: 8 * 32 is 256, and one byte holds none of it.
func TestWriteIdent(t *testing.T) {
	cases := []struct {
		name  string
		names int
		want  []byte
	}{
		{name: "the first name", names: 0, want: []byte{OpPush2, 0x00, 0x00, OpPush1, FRAME_POINTER, OpMemoryLoad, OpAdd}},
		{name: "the second", names: 1, want: []byte{OpPush2, 0x00, 0x20, OpPush1, FRAME_POINTER, OpMemoryLoad, OpAdd}},
		{name: "the ninth, which used to land on the first", names: 8, want: []byte{OpPush2, 0x01, 0x00, OpPush1, FRAME_POINTER, OpMemoryLoad, OpAdd}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			names := map[string]int{"x": tc.names * MEMORY_SLOT_SIZE}

			bs := bytes.NewBuffer(make([]byte, 0))
			if _, err := WriteIdent(bs, names, []byte("x")); err != nil {
				t.Fatalf("writing the binding: %v", err)
			}

			expected := append(append([]byte{}, tc.want...), OpMemoryStore)
			if got := bs.Bytes(); !bytes.Equal(got, expected) {
				t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
			}
		})
	}
}

func TestWriteLoad(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteLoad(bs, map[string]int{"test": 0x120}, []byte("test")); err != nil {
		t.Fatalf("writing the read: %v", err)
	}

	expected := []byte{OpPush2, 0x01, 0x20, OpPush1, FRAME_POINTER, OpMemoryLoad, OpAdd, OpMemoryLoad}
	if got := bs.Bytes(); !bytes.Equal(got, expected) {
		t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

// An argument arrives as a whole 32-byte word and is cut to the tape on the way in, the same
// as the evaluator does when it narrows the arguments it was handed.
// Reading one of the values applied to the scope reads the frame, and never the calldata.
// Whoever entered the scope put them there — the way in from a transaction copies them out of
// the calldata, and a scope calling another writes what it worked out — which is the whole of
// what the frame buys: a body that does not know how it was entered.
func TestWriteGetArg(t *testing.T) {
	cases := []struct {
		name string
		at   uint64
		want []byte
	}{
		{name: "the first position", at: 0, want: []byte{OpPush2, 0x00, 0x00}},
		{name: "the third", at: 2, want: []byte{OpPush2, 0x00, 0x40}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bs := bytes.NewBuffer(make([]byte, 0))
			if _, err := WriteGetArg(bs, byteutil.FromUint64(tc.at), byteutil.DefaultTapeSize); err != nil {
				t.Fatalf("writing the read: %v", err)
			}

			expected := append(append([]byte{}, tc.want...), OpPush1, FRAME_POINTER, OpMemoryLoad, OpAdd, OpMemoryLoad)
			if got := bs.Bytes(); !bytes.Equal(got, expected) {
				t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
			}
		})
	}
}

// Ending a scope goes back to whoever called it: the value is on the stack and the address to
// go back to is under it. It never returns to the chain — that is the epilogue's, and keeping it
// there is what lets one body serve a transaction and a scope alike.
func TestWriteReturnToCaller(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteReturnToCaller(bs); err != nil {
		t.Fatalf("writing the return: %v", err)
	}
	if got, want := bs.Bytes(), []byte{OpSwap1, OpJump}; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(want))
	}
}

// Answering the chain goes through a slot of its own, not the first one: that holds where the
// running scope's frame begins, and writing the answer there would lose the frame in the act
// of answering.
func TestWriteReturnToChain(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteReturnToChain(bs, 0, 8); err != nil {
		t.Fatalf("writing the answer: %v", err)
	}
	expected := []byte{OpPush1, RETURN_SCRATCH, OpMemoryStore, OpPush1, 0x20, OpPush1, RETURN_SCRATCH, OpReturn}
	if got := bs.Bytes(); !bytes.Equal(got, expected) {
		t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

// A run does not go through that slot, because it is not in one: it is already in memory, and
// what is on the stack is where it starts. So it is handed back from there, as many bytes as
// it has tapes — the same run the evaluator returns, byte for byte.
func TestAReturnOfARunHandsBackTheRunItself(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteReturnToChain(bs, 5, 8); err != nil {
		t.Fatalf("writing the answer: %v", err)
	}
	// Forty bytes, which is more than a word — the size that used to be refused.
	expected := []byte{OpPush2, 0x00, 40, OpSwap1, OpReturn}
	if got := bs.Bytes(); !bytes.Equal(got, expected) {
		t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

// What a return measures is what it writes, for the reason every size here is measured: a
// table of sizes is a second description of the same bytes.
func TestWhatAReturnMeasuresIsWhatItWrites(t *testing.T) {
	for _, tapes := range []int{0, 1, 2, 5, 8} {
		var bs bytes.Buffer
		if _, err := WriteReturnToChain(&bs, tapes, 8); err != nil {
			t.Fatalf("writing for %d tapes: %v", tapes, err)
		}
		measured, err := ReturnToChainSize(tapes, 8)
		if err != nil {
			t.Fatalf("measuring for %d tapes: %v", tapes, err)
		}
		if measured != bs.Len() {
			t.Errorf("%d tapes: it writes %d bytes and says %d", tapes, bs.Len(), measured)
		}
	}
}

// One entry of the dispatcher: read the selector out of the calldata, compare it with the one
// this scope answers to, jump to the body when they match. The address goes in two bytes, so
// a body past byte 255 of the runtime can be named.
func TestWriteDispatcher(t *testing.T) {
	cases := []struct {
		name   string
		jumpTo int
		want   []byte
	}{
		{name: "a body near the top", jumpTo: 10, want: []byte{OpPush2, 0x00, 0x0a}},
		{name: "a body past a byte", jumpTo: 300, want: []byte{OpPush2, 0x01, 0x2c}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bs := bytes.NewBuffer(make([]byte, 0))
			if _, err := WriteDispatcher(bs, "test", tc.jumpTo); err != nil {
				t.Fatalf("writing the dispatcher: %v", err)
			}

			expected := []byte{OpPush1, 0x00, OpCallDataLoad, OpPush1, 0xe0, OpShiftRight,
				OpPush4, 0x9c, 0x22, 0xff, 0x5f, OpEqual}
			expected = append(expected, tc.want...)
			expected = append(expected, OpJumpIf)

			got := bs.Bytes()
			if !bytes.Equal(got, expected) {
				t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
			}
			if len(got) != DISPATCHER_BYTES_SIZE {
				t.Errorf("it measures %d and the offsets are counted from %d", len(got), DISPATCHER_BYTES_SIZE)
			}
		})
	}
}

func TestWriteBodyCode(t *testing.T) {
	cases := []struct {
		Name        string
		Dispatchers []Dispatcher
		Root        *bytes.Buffer
		FnExpected  func() []byte
	}{
		{
			"sample_body_code_1",
			[]Dispatcher{
				{
					Selector: []byte("test"),
					Code:     bytes.NewBuffer([]byte{1}),
					Offset:   0,
					Length:   1,
				},
			},
			nil,
			func() []byte {
				return []byte{1}
			},
		},
		{
			"sample_body_code_2",
			[]Dispatcher{
				{
					Selector: []byte("test"),
					Code:     bytes.NewBuffer([]byte{1}),
					Offset:   0,
					Length:   1,
				},
			},
			bytes.NewBuffer([]byte{2}),
			func() []byte {
				return []byte{1, 2}
			},
		},
	}

	for _, c := range cases {
		bs := bytes.NewBuffer(make([]byte, 0))
		if _, err := WriteBodyCode(bs, c.Dispatchers, c.Root); err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		if got, expected := bs.Bytes(), c.FnExpected(); !bytes.Equal(got, expected) {
			t.Errorf("EVM body code: name: %v, got: %v, expected: %v", c.Name, byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
		}
	}
}

// A tape of N bytes holds values modulo 2^(8N), and the EVM wraps at 2^256 — so every result
// that could have left the width is cut back to it. At the full width there is nothing to
// cut, and the common case pays nothing.
func TestWriteMask(t *testing.T) {
	cases := []struct {
		name string
		size int
		want []byte
	}{
		{name: "one byte", size: 1, want: []byte{OpPush1, 0xff, OpAnd}},
		{name: "two bytes", size: 2, want: []byte{OpPush2, 0xff, 0xff, OpAnd}},
		{
			name: "the default",
			size: byteutil.DefaultTapeSize,
			want: []byte{OpPush8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, OpAnd},
		},
		{name: "the full word masks nothing", size: byteutil.MaxTapeSize, want: []byte{}},
		// Zero means unset, which is the default width rather than a mask of nothing.
		{
			name: "unset",
			size: 0,
			want: []byte{OpPush8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, OpAnd},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bs := bytes.NewBuffer(make([]byte, 0))
			if _, err := WriteMask(bs, tc.size); err != nil {
				t.Fatalf("WriteMask: %v", err)
			}
			if got := bs.Bytes(); !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// A call carries its own landing rather than jumping to the instruction after it: the address
// it pushes to come back to is a JUMPDEST inside its own bytes. It is worked out by measuring
// what it is about to write, which is why it stays right when what a call writes changes.
func TestACallComesBackToItsOwnJumpDestiny(t *testing.T) {
	const at = 100

	bs := bytes.NewBuffer(make([]byte, 0))
	inst := ir.NewInstructionOver([]byte("00"), ir.OpCall, ir.NameOf("f"), ir.Imm(1, 8))
	if err := WriteCall(bs, inst, ScopeOf(nil, nil, 8, map[string]Entry{"f": {At: 0x1234}}, false), at); err != nil {
		t.Fatalf("writing the call: %v", err)
	}

	code := bs.Bytes()
	landing := bytes.IndexByte(code, OpJumpDestiny)
	if landing < 0 {
		t.Fatal("a call went somewhere and left nowhere to come back to")
	}
	back := at + landing
	if want := []byte{OpPush2, byte(back >> 8), byte(back)}; !bytes.Contains(code, want) {
		t.Errorf("it comes back to %#x, and never pushes that address: %v", back, byteutil.ToUpperHex(code))
	}
	if want := []byte{OpPush2, 0x12, 0x34, OpJump}; !bytes.Contains(code, want) {
		t.Errorf("it does not go to the scope it names: %v", byteutil.ToUpperHex(code))
	}
}

// Only a scope bound at the top of a program is something a call can jump to. A call to
// anything else is refused rather than written: what would be written is a jump to an address
// no scope has, and that contract deploys and reverts when the call is reached.
func TestACallToSomethingThatIsNotAScopeIsRefused(t *testing.T) {
	inst := ir.NewInstructionOver([]byte("00"), ir.OpCall, ir.NameOf("nowhere"), ir.Imm(1, 8))

	err := WriteCall(io.Discard, inst, ScopeOf(nil, nil, 8, map[string]Entry{"elsewhere": {At: 0x40}}, false), 0)
	if err == nil {
		t.Fatal("it wrote a call to a scope that does not exist")
	}
	if !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("it says %q, and never names what was called", err)
	}
}

// A run wider than a word reaches the bytecode, because it never goes in a word.
//
// Five tapes of eight is forty bytes, and that used to be refused rather than written — a
// program that ran off a chain and could not reach one, which is the one thing this backend
// may not do. The tapes are in memory now, so how many there are is only a question of how
// much room the run gets.
func TestARunWiderThanAWordIsWritten(t *testing.T) {
	tapes := make([]ir.Operand, 0, 8)
	for at := 0; at < 8; at++ {
		tapes = append(tapes, ir.Imm(uint64(at), 8))
	}

	for _, count := range []int{1, 4, 5, 8} {
		inst := ir.NewInstructionOver([]byte("00"), ir.OpJoin, tapes[:count]...)
		if err := WriteJoin(io.Discard, inst, 8, RunLeadIn(8)); err != nil {
			t.Errorf("a run of %d tapes: %v", count, err)
		}
	}
}

// The room a run takes is its tapes, and what a word has over a tape in front of them.
func TestHowMuchRoomARunTakes(t *testing.T) {
	for _, tc := range []struct {
		tapes, tapeSize, want int
	}{
		{tapes: 1, tapeSize: 8, want: 24 + 8},
		{tapes: 4, tapeSize: 8, want: 24 + 32},
		{tapes: 5, tapeSize: 8, want: 24 + 40},
		// A tape as wide as a word needs no room in front: the word is the tape.
		{tapes: 2, tapeSize: 32, want: 64},
	} {
		if got := RunRoom(tc.tapes, tc.tapeSize); got != tc.want {
			t.Errorf("%d tapes of %d take %d, want %d", tc.tapes, tc.tapeSize, got, tc.want)
		}
	}
}

// A field counts forward from where the run starts, and how many tapes the run has does not
// come into it.
//
// It used to count back from the end, because the run was a word and a field was a shift: the
// first of two was shifted down by a tape, the first of three by two, so the same index read a
// different place depending on how long the run was. In memory it is an address — the index
// times a tape from the start — and the run's length is only about where it ends.
func TestAFieldCountsForwardFromTheStartOfTheRun(t *testing.T) {
	for _, tc := range []struct {
		name  string
		at    uint64
		tapes uint64
		want  []byte
	}{
		{name: "the first of two", at: 0, tapes: 2, want: []byte{OpMemoryLoad, OpPush1, 192, OpShiftRight}},
		{name: "the last of two", at: 1, tapes: 2, want: []byte{OpPush2, 0, 8, OpAdd, OpMemoryLoad, OpPush1, 192, OpShiftRight}},
		// The same index of a longer run reads the same place, which is the whole difference.
		{name: "the middle of three", at: 1, tapes: 3, want: []byte{OpPush2, 0, 8, OpAdd, OpMemoryLoad, OpPush1, 192, OpShiftRight}},
		{name: "the last of three", at: 2, tapes: 3, want: []byte{OpPush2, 0, 16, OpAdd, OpMemoryLoad, OpPush1, 192, OpShiftRight}},
		// Past four tapes, which is where a run stopped reaching the bytecode at all.
		{name: "the seventh of eight", at: 6, tapes: 8, want: []byte{OpPush2, 0, 48, OpAdd, OpMemoryLoad, OpPush1, 192, OpShiftRight}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bs := bytes.NewBuffer(make([]byte, 0))
			inst := ir.NewInstructionOver([]byte("00"), ir.OpField,
				ir.RefTo([]byte("01")), ir.Const(tc.at, 8), ir.Const(tc.tapes, 8))
			if err := WriteField(bs, inst, 8); err != nil {
				t.Fatalf("writing the field: %v", err)
			}
			if got := bs.Bytes(); !bytes.Equal(got, tc.want) {
				t.Errorf("it writes %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(tc.want))
			}
		})
	}
}

// A tape as wide as a word is the word, so a field of it is a read and nothing after it.
func TestAFieldOfAWordWideTapeIsJustTheRead(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	inst := ir.NewInstructionOver([]byte("00"), ir.OpField,
		ir.RefTo([]byte("01")), ir.Const(1, 32), ir.Const(2, 32))
	if err := WriteField(bs, inst, 32); err != nil {
		t.Fatalf("writing the field: %v", err)
	}
	want := []byte{OpPush2, 0, 32, OpAdd, OpMemoryLoad}
	if got := bs.Bytes(); !bytes.Equal(got, want) {
		t.Errorf("it writes %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(want))
	}
}

// A tape built out of values a program works out would need each of them brought to the top of
// the stack in turn, and that is not written. It is refused rather than written wrong: the
// values are on the stack in the order they appear and the fold needs them in the other one.
func TestATapeBuiltFromWorkedOutValuesIsRefused(t *testing.T) {
	inst := ir.NewInstructionOver([]byte("00"), ir.OpPull,
		ir.RefTo([]byte("01")), ir.RefTo([]byte("02")), ir.RefTo([]byte("03")))

	err := WriteTapePull(io.Discard, inst, 8)
	if err == nil {
		t.Fatal("it wrote a fold the stack cannot give it")
	}
	if !strings.Contains(err.Error(), "one at a time") {
		t.Errorf("it says %q, and never says what to do instead", err)
	}
}

// A tape literal collapses into one shift and one or, however many values were written between
// the brackets: their lengths are known while compiling, so the whole run of them is a number.
func TestATapeLiteralIsOneShiftAndOneOr(t *testing.T) {
	items := []ir.Operand{ir.RefTo([]byte("01"))}
	for _, value := range []uint64{1, 2, 3} {
		items = append(items, ir.Imm(value, 8))
	}

	bs := bytes.NewBuffer(make([]byte, 0))
	if err := WriteTapePull(bs, ir.NewInstructionOver([]byte("00"), ir.OpPull, items...), 8); err != nil {
		t.Fatalf("writing the literal: %v", err)
	}

	// Three values of one byte each move the tape three places, and what enters is the three
	// of them read as one number. The mask after it is what keeps the tape its own width,
	// which is what makes whatever reached the left end fall off.
	expected := []byte{OpPush2, 0x00, 24, OpShiftLeft, OpPush3, 0x01, 0x02, 0x03, OpOr,
		OpPush8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, OpAnd}
	if got := bs.Bytes(); !bytes.Equal(got, expected) {
		t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

// How much of a value means something is worked out from the word itself, by halving: five
// times over, the value is asked whether anything survives a shift of sixteen bytes, then
// eight, four, two, one. Nothing branches, so the same run of instructions answers for every
// word — and the last byte is added at the end and never asked about, which is what makes a
// value of zero one byte long rather than none.
func TestWriteSignificantLengthAsksFiveTimesAndNeverBranches(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if err := WriteSignificantLength(bs); err != nil {
		t.Fatalf("writing the length: %v", err)
	}

	code := bs.Bytes()
	for _, jump := range []byte{OpJump, OpJumpIf, OpJumpDestiny} {
		if bytes.IndexByte(code, jump) >= 0 {
			t.Errorf("it branches, and a run of instructions that branches is five jumps: %v",
				byteutil.ToUpperHex(code))
		}
	}

	// One shift asked about per step, and each is a whole number of bytes.
	for _, step := range []byte{128, 64, 32, 16, 8} {
		if !bytes.Contains(code, []byte{OpPush1, step, OpShiftRight}) {
			t.Errorf("it never asks whether anything survives a shift of %d bits", step)
		}
	}
	// And the byte that always counts.
	if !bytes.Contains(code, []byte{OpPop, OpPush1, BYTE_SIZE, OpAdd}) {
		t.Error("it does not add the byte that is never asked about")
	}
}

// A tape literal collapses because the lengths of what enters it are known while compiling.
// When they are not — a value the program works out — the length is worked out beside it, and
// the tape and the value change places around it.
func TestWriteTapePullWorksOutALengthItCannotKnow(t *testing.T) {
	inst := ir.NewInstructionOver([]byte("00"), ir.OpPull, ir.RefTo([]byte("01")), ir.RefTo([]byte("02")))

	bs := bytes.NewBuffer(make([]byte, 0))
	if err := WriteTapePull(bs, inst, 8); err != nil {
		t.Fatalf("writing the pull: %v", err)
	}

	code := bs.Bytes()
	if code[0] != OpDup1 {
		t.Errorf("it opens with %v, want it keeping the value before measuring it",
			byteutil.ToUpperHex(code[:1]))
	}
	if !bytes.Contains(code, []byte{OpShiftLeft, OpOr}) {
		t.Errorf("it does not move the tape over and let the value in: %v", byteutil.ToUpperHex(code))
	}
}

// Pushing lets a value in at the left, so the tape moves down by the value's own length and the
// value moves up by what is left of the tape's width. A value is a tape, so it is never wider
// than one and neither shift can run backwards.
func TestWriteTapePushMovesBothWays(t *testing.T) {
	written := ir.NewInstruction([]byte("00"), ir.OpPush, ir.RefTo([]byte("01")), ir.Imm(5, 8))

	bs := bytes.NewBuffer(make([]byte, 0))
	if err := WriteTapePush(bs, written, 8); err != nil {
		t.Fatalf("writing the push: %v", err)
	}
	// One byte of value: the tape moves down eight bits, and the value goes in at the top.
	if want := []byte{OpPush2, 0x00, 8, OpShiftRight}; !bytes.HasPrefix(bs.Bytes(), want) {
		t.Errorf("it opens with %v, want %v", byteutil.ToUpperHex(bs.Bytes()[:4]), byteutil.ToUpperHex(want))
	}

	worked := ir.NewInstruction([]byte("00"), ir.OpPush, ir.RefTo([]byte("01")), ir.RefTo([]byte("02")))
	bs = bytes.NewBuffer(make([]byte, 0))
	if err := WriteTapePush(bs, worked, 8); err != nil {
		t.Fatalf("writing the push: %v", err)
	}
	if !bytes.Contains(bs.Bytes(), []byte{OpSub, OpShiftLeft, OpOr}) {
		t.Errorf("it does not take the value's length off the width: %v", byteutil.ToUpperHex(bs.Bytes()))
	}
}

// head keeps the first bytes of what a tape says and tail drops them, and both count in
// significant bytes — so both begin by asking how long the value is, and what is kept is
// measured from the other end.
func TestWriteTapeHeadAndTailCountFromTheEnd(t *testing.T) {
	inst := ir.NewInstruction([]byte("00"), ir.OpHead, ir.RefTo([]byte("01")), ir.Const(2, 8))

	head := bytes.NewBuffer(make([]byte, 0))
	if err := WriteTapeHead(head, inst, 8); err != nil {
		t.Fatalf("writing the head: %v", err)
	}
	if got := head.Bytes(); got[len(got)-1] != OpShiftRight {
		t.Errorf("keeping the first bytes does not end by moving the tape down: %v",
			byteutil.ToUpperHex(got))
	}

	tail := bytes.NewBuffer(make([]byte, 0))
	if err := WriteTapeTail(tail, ir.NewInstruction([]byte("00"), ir.OpTail, ir.RefTo([]byte("01")), ir.Const(2, 8)), 8); err != nil {
		t.Fatalf("writing the tail: %v", err)
	}
	if got := tail.Bytes(); got[len(got)-1] != OpAnd {
		t.Errorf("dropping the first bytes does not end by keeping the rest: %v",
			byteutil.ToUpperHex(got))
	}

	// An index of nothing keeps everything, and asks about no index at all.
	whole := bytes.NewBuffer(make([]byte, 0))
	if err := WriteTapeTail(whole, ir.NewInstruction([]byte("00"), ir.OpTail, ir.RefTo([]byte("01")), ir.Const(0, 8)), 8); err != nil {
		t.Fatalf("writing the tail: %v", err)
	}
	if bytes.Contains(whole.Bytes(), []byte{OpGreaterThan}) {
		t.Error("it holds an index of nothing to the value's length, and there is nothing to hold")
	}
}
