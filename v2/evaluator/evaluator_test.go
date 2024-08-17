package evaluator

import "testing"

func TestIsLabel(t *testing.T) {
	values := [][]byte{
		[]byte("0t"),
		[]byte("-1t"),
	}
	for _, v := range values {
		if !IsTemp(v) {
			t.Errorf("unrecognized as label pattern, got: %s", v)
			return
		}
	}
}
