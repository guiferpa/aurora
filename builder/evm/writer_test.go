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
			manager := NewIdentManager()
			for i := 0; i < tc.names; i++ {
				manager.SetOffset(string(rune('a'+i)), i*MEMORY_SLOT_SIZE)
			}

			bs := bytes.NewBuffer(make([]byte, 0))
			if _, err := WriteIdent(bs, manager, []byte("x")); err != nil {
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
	manager := NewIdentManager()
	manager.SetOffset("test", 0x120)

	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteLoad(bs, manager, []byte("test")); err != nil {
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
// go back to is under it. It never answers the chain — that is the epilogue's, and keeping it
// there is what lets one body serve a transaction and a scope alike.
func TestWriteReturn(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteReturn(bs); err != nil {
		t.Fatalf("writing the return: %v", err)
	}
	if got, want := bs.Bytes(), []byte{OpSwap1, OpJump}; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(want))
	}
}

// Answering the chain goes through a slot of its own, not the first one: that holds where the
// running scope's frame begins, and writing the answer there would lose the frame in the act
// of answering.
func TestWriteAnswer(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteAnswer(bs); err != nil {
		t.Fatalf("writing the answer: %v", err)
	}
	expected := []byte{OpPush1, RETURN_SCRATCH, OpMemoryStore, OpPush1, 0x20, OpPush1, RETURN_SCRATCH, OpReturn}
	if got := bs.Bytes(); !bytes.Equal(got, expected) {
		t.Errorf("got %v, want %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
	if bs.Len() != ANSWER_SIZE {
		t.Errorf("it measures %d and says it measures %d", bs.Len(), ANSWER_SIZE)
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

// Measuring is writing to something that only counts, so what it says and what comes out
// cannot disagree. That is the whole reason it is done this way rather than with a table of
// sizes: a table is a second description, and two descriptions of one thing drift.
func TestWhatIsMeasuredIsWhatIsWritten(t *testing.T) {
	insts := []ir.Instruction{
		ir.NewInstruction([]byte("00"), ir.OpGetFeed, ir.Const(0, 8), ir.Nothing()),
		ir.NewInstruction([]byte("01"), ir.OpSave, ir.Imm(10, 8), ir.Nothing()),
		ir.NewInstruction([]byte("02"), ir.OpAdd, ir.RefTo([]byte("00")), ir.RefTo([]byte("01"))),
		ir.NewInstruction([]byte("03"), ir.OpIdent, ir.NameOf("x"), ir.RefTo([]byte("02"))),
		ir.NewInstruction([]byte("04"), ir.OpLoad, ir.NameOf("x"), ir.Nothing()),
	}

	positions, err := PositionsOf(insts, nil, ScopeOf(insts, 8, nil, true))
	if err != nil {
		t.Fatalf("measuring: %v", err)
	}
	if len(positions) != len(insts)+1 {
		t.Fatalf("measured %d positions for %d instructions", len(positions), len(insts))
	}

	bs := bytes.NewBuffer(make([]byte, 0))
	im := NewIdentManager()
	for at, inst := range insts {
		before := bs.Len()
		if err := WriteInstruction(bs, im, inst, 0, ScopeOf(insts, 8, nil, true)); err != nil {
			t.Fatalf("writing: %v", err)
		}
		if want := positions[at+1] - positions[at]; bs.Len()-before != want {
			t.Errorf("%s measured %d bytes and wrote %d",
				ir.ResolveOpCode(inst.GetOpCode()), want, bs.Len()-before)
		}
	}
}

// A call carries its own landing rather than jumping to the instruction after it: the address
// it pushes to come back to is a JUMPDEST inside its own bytes. It is worked out by measuring
// what it is about to write, which is why it stays right when what a call writes changes.
func TestACallComesBackToItsOwnJumpDestiny(t *testing.T) {
	const at = 100

	bs := bytes.NewBuffer(make([]byte, 0))
	inst := ir.NewInstructionOver([]byte("00"), ir.OpCall, ir.NameOf("f"), ir.Imm(1, 8))
	if err := WriteCall(bs, inst, ScopeOf(nil, 8, map[string]int{"f": 0x1234}, false), at); err != nil {
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

	err := WriteCall(io.Discard, inst, ScopeOf(nil, 8, map[string]int{"elsewhere": 0x40}, false), 0)
	if err == nil {
		t.Fatal("it wrote a call to a scope that does not exist")
	}
	if !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("it says %q, and never names what was called", err)
	}
}
