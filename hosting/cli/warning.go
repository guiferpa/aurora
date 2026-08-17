package cli

import (
	"io"

	"github.com/fatih/color"

	"github.com/guiferpa/aurora/wire/diag"
)

// ReportWarnings prints what the compiler and the backend want to say about a program without
// refusing it. They go to stderr, so a pipeline reading the program's output is unaffected.
//
// The two used to arrive as two types — one owned by the emitter, one by the backend — and
// this file carried a pair of functions to make them into a third. They speak the same shape
// now, which is what a warning was all along: a message, and sometimes a place.
//
// A warning that points at a place in the source is written as file:line:column, which is
// what an editor follows to jump there.
func ReportWarnings(w io.Writer, source string, warnings []diag.Warning) {
	if w == nil {
		return
	}
	yellow := color.New(color.FgHiYellow)
	for _, warning := range warnings {
		if warning.Positioned() {
			_, _ = yellow.Fprintf(w, "%s:%d:%d: warning: %s\n", source, warning.Line, warning.Column, warning)
			continue
		}
		_, _ = yellow.Fprintf(w, "%s: warning: %s\n", source, warning)
	}
}
