package evaluator

import "testing"

func TestIsLabel(t *testing.T) {
	values := [][]byte{
		[]byte("0t"),
		[]byte("0-1t"),
	}
	for _, v := range values {
		if !isTemp(v) {
			t.Errorf("unrecognized as label pattern, got: %s", v)
			return
		}
	}
}
