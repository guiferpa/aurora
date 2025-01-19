package environ

import (
	"bytes"
	"testing"
)

func TestPoolIsEmpty(t *testing.T) {
	p := NewPool(New(nil))
	if p.IsEmpty() {
		t.Error("unexpected empty as result")
	}
	if env := p.Current(); env == nil {
		t.Error("unexpected current env as nil value")
	}
}

func TestPool(t *testing.T) {
	p := NewPool(New(nil))

	k := "A"
	v := bytes.NewBufferString("B")
	p.SetLocal(k, v.Bytes())
	got := p.Current().GetLocal(k)
	if !bytes.Equal(got, v.Bytes()) {
		t.Errorf("unexpected result: got: %v, expected: %v", got, v.Bytes())
		return
	}

	k = "C"
	v = bytes.NewBufferString("D")
	p.SetLocal(k, v.Bytes())
	got = p.Current().GetLocal(k)
	if !bytes.Equal(got, v.Bytes()) {
		t.Errorf("unexpected result: got: %v, expected: %v", got, v.Bytes())
		return
	}

	p.Ahead()
	got = p.Current().GetLocal(k)
	if got != nil {
		t.Errorf("unexpected result: got: %v, expected: nil", got)
		return
	}

	got = p.QueryLocal(k)
	if got == nil {
		t.Errorf("unexpected result: got: %v, expected: %v", got, v.Bytes())
		return
	}
	if !bytes.Equal(got, v.Bytes()) {
		t.Errorf("unexpected result: got: %v, expected: %v", got, v.Bytes())
		return
	}
}
