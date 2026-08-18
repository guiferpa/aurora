package builtin

import (
	"strconv"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
)

func TestAssertFunction(t *testing.T) {
	passed, err := AssertFunction(byteutil.TrueTape(8), "unused")
	if !passed || err != nil {
		t.Errorf("a true condition passes: got (%v, %v)", passed, err)
	}

	passed, err = AssertFunction(byteutil.FalseTape(8), "boom")
	if passed {
		t.Error("a false condition fails")
	}
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %v, want it to carry the message", err)
	}
}

// A failing assertion reads its message the same way printc does. Walking it with a stride
// of eight regardless of the tape size read past the end of a narrow reel and brought the
// whole program down with it.
func TestAssertFunctionWithNarrowTapes(t *testing.T) {
	for _, size := range []int{1, 2, 4, 8, 32} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			_, err := AssertFunction(byteutil.FalseTape(size), "boom")
			if err == nil {
				t.Fatal("expected a failure")
			}
			if !strings.Contains(err.Error(), "boom") {
				t.Errorf("error = %q, want it to carry the message", err)
			}
		})
	}
}

func TestAssertFunctionWithNoMessage(t *testing.T) {
	_, err := AssertFunction(byteutil.FalseTape(8), "")
	if err == nil {
		t.Fatal("expected a failure")
	}
	if !strings.Contains(err.Error(), "assertion failed") {
		t.Errorf("error = %q, want it to still say what happened", err)
	}
}
