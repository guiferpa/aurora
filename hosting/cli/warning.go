package cli

import (
	"io"

	"github.com/fatih/color"

	"github.com/guiferpa/aurora/builder/evm"
	"github.com/guiferpa/aurora/emitter"
)

// A Warning is something worth saying about a program that is not a reason to refuse it.
//
// The compiler and the backend each answer in their own words — a phase does not speak for
// the one after it — so this is where the two are made into one thing to print. Line and
// column are zero when the warning is about the program as a whole rather than a place in it.
type Warning struct {
	Message string
	Line    int
	Column  int
}

// CompilerWarnings and BackendWarnings carry each phase's answer across without either of
// them having to know what the other calls a warning.
func CompilerWarnings(warnings []emitter.Warning) []Warning {
	out := make([]Warning, 0, len(warnings))
	for _, warning := range warnings {
		out = append(out, Warning{Message: warning.Message, Line: warning.Line, Column: warning.Column})
	}
	return out
}

func BackendWarnings(warnings []evm.Warning) []Warning {
	out := make([]Warning, 0, len(warnings))
	for _, warning := range warnings {
		out = append(out, Warning{Message: warning.Message})
	}
	return out
}

// ReportWarnings prints what the compiler and the backend want to say about a program without
// refusing it. They go to stderr, so a pipeline reading the program's output is unaffected.
//
// A warning that points at a place in the source is written as file:line:column, which is
// what an editor follows to jump there.
func ReportWarnings(w io.Writer, source string, warnings []Warning) {
	if w == nil {
		return
	}
	yellow := color.New(color.FgHiYellow)
	for _, warning := range warnings {
		if warning.Line > 0 {
			_, _ = yellow.Fprintf(w, "%s:%d:%d: warning: %s\n", source, warning.Line, warning.Column, warning.Message)
			continue
		}
		_, _ = yellow.Fprintf(w, "%s: warning: %s\n", source, warning.Message)
	}
}
