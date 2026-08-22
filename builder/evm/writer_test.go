package evm

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
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

func TestWriteIdent(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	identManager := NewIdentManager()
	label := "test"
	offset := byte(0x20)
	identManager.SetOffset(label, offset)
	if _, err := WriteIdent(bs, identManager, []byte(label)); err != nil {
		t.Errorf("Error writing ident: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{OpPush1, offset, OpMemoryStore}
	if !bytes.Equal(got, expected) {
		t.Errorf("Ident: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

func TestWriteLoad(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	identManager := NewIdentManager()
	identManager.SetOffset("test", 0x20)
	left := []byte("test")
	if _, err := WriteLoad(bs, identManager, left); err != nil {
		t.Errorf("Error writing load: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{OpPush1, 0x20, OpMemoryLoad}
	if !bytes.Equal(got, expected) {
		t.Errorf("Load: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

// An argument arrives as a whole 32-byte word and is cut to the tape on the way in, the same
// as the evaluator does when it narrows the arguments it was handed.
func TestWriteGetArg(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	index := byteutil.FromUint64(0)
	if _, err := WriteGetArg(bs, index, byteutil.DefaultTapeSize); err != nil {
		t.Errorf("Error writing get arg: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{OpPush1, 0x20, OpCallDataLoad, OpPush8, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, OpAnd}
	if !bytes.Equal(got, expected) {
		t.Errorf("GetArg: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

func TestWriteReturn(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteReturn(bs); err != nil {
		t.Errorf("Error writing return: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{
		OpPush1, 0x00, OpMemoryStore, // store stack top at mem[0]
		OpPush1, 0x20, OpPush1, 0x00, OpReturn,
	}
	if !bytes.Equal(got, expected) {
		t.Errorf("Return: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
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
