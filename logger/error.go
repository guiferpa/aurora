package logger

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// CommandError reports a failed command and exits. Diagnostics go to stderr, so a shell
// pipeline reading a program's output does not swallow them.
func CommandError(err error) {
	if err == nil {
		return
	}
	_, _ = color.New(color.BgBlack, color.FgHiMagenta).Fprintln(os.Stderr, err)
	os.Exit(2)
}

func AssertError(errs []error, filename string) {
	if len(errs) == 0 {
		return
	}
	_, _ = color.New(color.FgWhite).Fprintln(os.Stderr, fmt.Sprintf("Assertion errors in %s:", filename))
	for _, err := range errs {
		_, _ = color.New(color.BgBlack, color.FgRed).Fprintln(os.Stderr, err)
	}
	os.Exit(3)
}
