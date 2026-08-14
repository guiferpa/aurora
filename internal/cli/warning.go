package cli

import (
	"io"

	"github.com/fatih/color"

	"github.com/guiferpa/aurora/emitter"
)

// ReportWarnings prints what the compiler wants to say about a program without refusing
// it. They go to stderr, so a pipeline reading the program's output is unaffected.
func ReportWarnings(w io.Writer, warnings []emitter.Warning) {
	if w == nil {
		return
	}
	for _, warning := range warnings {
		_, _ = color.New(color.FgHiYellow).Fprintf(w, "warning: %s\n", warning)
	}
}
