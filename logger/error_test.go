package logger

import (
	"errors"
	"strings"
	"testing"
)

// These used to run in a subprocess, because the package wrote to stderr and ended the
// process: the test re-executed itself, read both streams and checked the exit code. None of
// that is needed now that it spells a message and hands it back — which is the point of the
// change, seen from a test.

func TestCommandError(t *testing.T) {
	got := CommandError(errors.New("something went wrong"))

	if !strings.Contains(got, "something went wrong") {
		t.Errorf("spelled %q, want it to carry the error", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("spelled %q, want a line rather than a fragment", got)
	}
}

// A command that did not fail has nothing to say, and nothing is not an empty line: whoever
// writes this out would print a blank one.
func TestCommandErrorOfNothing(t *testing.T) {
	if got := CommandError(nil); got != "" {
		t.Errorf("spelled %q for no error at all", got)
	}
}
