package cli

import (
	"io"

	"github.com/fatih/color"

	"github.com/guiferpa/aurora/emitter"
)

// ReportWarnings prints what the compiler wants to say about a program without refusing
// it. They go to stderr, so a pipeline reading the program's output is unaffected.
//
// A warning that points at a place in the source is written as file:line:column, which is
// what an editor follows to jump there.
func ReportWarnings(w io.Writer, source string, warnings []emitter.Warning) {
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
