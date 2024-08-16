package environ

import (
	"bytes"
	"testing"
)

func TestEnvironSetter(t *testing.T) {
	k := "A"
	v := bytes.NewBufferString("B")
	env := New(nil)
	env.Set(k, v.Bytes())
	got, ok := env.table[k]
	if !ok {
		t.Errorf("unexpected emty result: got: %v, expected: %v", got, v.Bytes())
	}
	if bytes.Compare(got, v.Bytes()) != 0 {
		t.Errorf("unexpected result: got: %v, expected: %v", got, v.Bytes())
	}
}

func TestEnvironGetter(t *testing.T) {
	k := "A"
	v := bytes.NewBufferString("B")
	env := New(nil)
	env.Set(k, v.Bytes())
	got := env.Get(k)
	if bytes.Compare(got, v.Bytes()) != 0 {
		t.Errorf("unexpected result: got: %v, expected: %v", got, v.Bytes())
	}
}
