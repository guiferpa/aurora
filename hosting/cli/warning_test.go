package cli

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/wire/diag"
)

// This is the line a user reads when the compiler has something to say and compiles anyway.
// Nothing covered it: the builder's warnings were tested as a list of messages, and how they
// reach a person was not tested at all.

func TestReportWarningsWritesWhereToLook(t *testing.T) {
	out := &strings.Builder{}

	ReportWarnings(out, "main.ar", []diag.Warning{
		{Message: "257 deferred scopes in one scope", Line: 12, Column: 5},
	})

	got := out.String()
	// An editor follows file:line:column to jump there, so the shape is the feature.
	if !strings.Contains(got, "main.ar:12:5: warning: 257 deferred scopes in one scope") {
		t.Errorf("wrote %q, want it to point at the place in the source", got)
	}
}

// A backend warns about the program, not about a line: writing "main.ar:0:0" would send an
// editor to the top of the file for something that is not there.
func TestReportWarningsWithNoPlaceToPointAt(t *testing.T) {
	out := &strings.Builder{}

	ReportWarnings(out, "main.ar", []diag.Warning{{Message: "printd writes a log"}})

	got := out.String()
	if !strings.Contains(got, "main.ar: warning: printd writes a log") {
		t.Errorf("wrote %q, want the file and the message", got)
	}
	if strings.Contains(got, ":0:") {
		t.Errorf("wrote %q, want no place at all", got)
	}
}

// Every warning is written, in the order they were found: the first thing said is the first
// thing that went missing.
func TestReportWarningsWritesEveryOne(t *testing.T) {
	out := &strings.Builder{}

	ReportWarnings(out, "main.ar", []diag.Warning{
		{Message: "first"},
		{Message: "second", Line: 2, Column: 1},
	})

	got := out.String()
	if lines := strings.Count(strings.TrimSpace(got), "\n") + 1; lines != 2 {
		t.Errorf("wrote %d lines for two warnings: %q", lines, got)
	}
	if strings.Index(got, "first") > strings.Index(got, "second") {
		t.Errorf("wrote %q, want them in the order they were found", got)
	}
}

// Nowhere to write is not a failure: "aurora run" passes nil when nobody asked for warnings,
// and it must not fall over on a program that has some.
func TestReportWarningsWithNowhereToWrite(t *testing.T) {
	ReportWarnings(nil, "main.ar", []diag.Warning{{Message: "printd writes a log"}})
}

// Nothing to say is nothing written, rather than an empty line.
func TestReportWarningsWithNothingToSay(t *testing.T) {
	out := &strings.Builder{}

	ReportWarnings(out, "main.ar", nil)

	if got := out.String(); got != "" {
		t.Errorf("wrote %q for no warnings", got)
	}
}
