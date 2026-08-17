package emitter

import (
	"github.com/guiferpa/aurora/wire/diag"
	"strconv"
	"strings"
	"testing"
)

func warningsFor(t *testing.T, source string, tapeSize int) []diag.Warning {
	t.Helper()
	return compileWith(t, source, tapeSize).Warnings
}

// A scope holding more deferred scopes than its tape can index gets a warning, because the
// index wraps and a call reaches a different scope instead of failing. It is a warning and
// not an error: the count is static and cannot know how often a scope runs, and a program
// already running should never be stopped by it.
func TestDeferCapacityWarning(t *testing.T) {
	source := strings.Repeat("ident d = defer { 1; };\n", 300)
	// Every binding is named the same, which the evaluator would reject — the warning is a
	// compile-time count and does not care.

	warnings := warningsFor(t, source, 1)
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %v", warnings)
	}
	message := warnings[0].Message
	for _, want := range []string{"300", "1-byte", "256"} {
		if !strings.Contains(message, want) {
			t.Errorf("warning = %q, want it to mention %q", message, want)
		}
	}
}

func TestNoWarningWhenTheyFit(t *testing.T) {
	cases := []struct {
		name     string
		defers   int
		tapeSize int
	}{
		{name: "few defers, one byte", defers: 10, tapeSize: 1},
		{name: "exactly the capacity", defers: 256, tapeSize: 1},
		{name: "many defers, two bytes", defers: 300, tapeSize: 2},
		{name: "many defers, default tape", defers: 300, tapeSize: 8},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Repeat("ident d = defer { 1; };\n", tc.defers)
			if warnings := warningsFor(t, source, tc.tapeSize); len(warnings) != 0 {
				t.Errorf("expected no warnings, got %v", warnings)
			}
		})
	}
}

// The tally is per scope, since each running scope keeps its own. Defers spread across
// nested scopes do not add up into one.
func TestDeferCapacityCountsPerScope(t *testing.T) {
	var source strings.Builder
	for i := 0; i < 4; i++ {
		source.WriteString("{\n")
		source.WriteString(strings.Repeat("ident d = defer { 1; };\n", 100))
		source.WriteString("};\n")
	}

	if warnings := warningsFor(t, source.String(), 1); len(warnings) != 0 {
		t.Errorf("400 defers across four scopes fit: %v", warnings)
	}
}

func TestDeferCapacityWarnsInsideANestedScope(t *testing.T) {
	source := "{\n" + strings.Repeat("ident d = defer { 1; };\n", 300) + "};\n"

	if warnings := warningsFor(t, source, 1); len(warnings) != 1 {
		t.Errorf("expected the inner scope to be counted, got %v", warnings)
	}
}

func TestWarningsAreReportedPerTapeSize(t *testing.T) {
	source := strings.Repeat("ident d = defer { 1; };\n", 70000)
	// 70000 exceeds a two-byte tape (65536) but fits anything wider.
	for _, size := range []int{1, 2} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			if warnings := warningsFor(t, source, size); len(warnings) == 0 {
				t.Error("expected a warning")
			}
		})
	}
	if warnings := warningsFor(t, source, 4); len(warnings) != 0 {
		t.Errorf("a four-byte tape names plenty: %v", warnings)
	}
}
