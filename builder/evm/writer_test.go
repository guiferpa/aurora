package evm

import (
	"bytes"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
)

func TestWriteInstantiateBlock(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	runtimeSize := byte(8)
	if _, err := WriteInstantiateBlock(bs, runtimeSize); err != nil {
		t.Errorf("Error writing instantiate block: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{
		OpPush1, runtimeSize,
		OpPush1, 0x0c,
		OpPush1, 0x00,
		OpCodeCopy,
		OpPush1, runtimeSize,
		OpPush1, 0x00,
		OpReturn,
	}
	if !bytes.Equal(got, expected) {
		t.Errorf("Instantiate block: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
	}
}

func TestWriteBool(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteBool(bs, byteutil.True[0]); err != nil {
		t.Errorf("Error writing bool: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{OpPush1, 1}
	if !bytes.Equal(got, expected) {
		t.Errorf("Bool: got: %v, expected: %v", byteutil.ToUpperHex(got), byteutil.ToUpperHex(expected))
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
	if _, err := WriteSave(bs, operand); err != nil {
		t.Errorf("Error writing save: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{OpPush1, 1} // single byte: PUSH1 only
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

func TestWriteGetArg(t *testing.T) {
	bs := bytes.NewBuffer(make([]byte, 0))
	index := byteutil.FromUint64(0)
	if _, err := WriteGetArg(bs, index); err != nil {
		t.Errorf("Error writing get arg: %v", err)
		return
	}
	got := bs.Bytes()
	expected := []byte{OpPush1, 0x20, OpCallDataLoad}
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

func TestWriteDispatcher(t *testing.T) {
	cases := []struct {
		Name       string
		FnExpected func() []byte
	}{
		{
			"sample_dispatcher_1",
			func() []byte {
				expected := []byte{OpPush1, 0x00}
				expected = append(expected, OpCallDataLoad)
				expected = append(expected, []byte{OpPush1, 0xe0}...)
				expected = append(expected, OpShiftRight)
				expected = append(expected, []byte{OpPush4, 0x9c, 0x22, 0xff, 0x5f}...)
				expected = append(expected, OpEqual)
				expected = append(expected, []byte{OpPush1, 0x0a}...)
				expected = append(expected, OpJumpIf)
				return expected
			},
		},
	}

	for _, c := range cases {
		bs := bytes.NewBuffer(make([]byte, 0))
		if _, err := WriteDispatcher(bs, "test", 10); err != nil {
			t.Errorf("%v: %v", c.Name, err)
			return
		}
		if got, expected := bs.Bytes(), c.FnExpected(); !bytes.Equal(got, expected) {
			t.Errorf("EVM dispatcher: name: %v, got: %v, expected: %v", c.Name, byteutil.ToUpperHex(got), byteutil.ToUpperHex(c.FnExpected()))
		}
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
