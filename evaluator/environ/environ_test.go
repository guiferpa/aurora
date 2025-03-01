package environ

import (
	"bytes"
	"testing"
)

func TestEnvironSetter(t *testing.T) {
	k := "A"
	v := bytes.NewBufferString("B")
	env := New(nil)
	env.SetLocaL(k, v.Bytes())
	got, ok := env.table[k]
	if !ok {
		t.Errorf("unexpected emty result: got: %v, expected: %v", got, v.Bytes())
	}
	if !bytes.Equal(got, v.Bytes()) {
		t.Errorf("unexpected result: got: %v, expected: %v", got, v.Bytes())
	}
}

func TestEnvironGetter(t *testing.T) {
	k := "A"
	v := bytes.NewBufferString("B")
	env := New(nil)
	env.SetLocaL(k, v.Bytes())
	got := env.GetLocal(k)
	if !bytes.Equal(got, v.Bytes()) {
		t.Errorf("unexpected result: got: %v, expected: %v", got, v.Bytes())
	}
}
