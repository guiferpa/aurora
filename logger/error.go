package logger

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

func CommandError(err error) {
	if err == nil {
		return
	}
	_, _ = color.New(color.BgBlack, color.FgHiMagenta).Println(err)
	os.Exit(2)
}

func AssertError(errs []error, filename string) {
	if len(errs) == 0 {
		return
	}
	_, _ = color.New(color.FgWhite).Println(fmt.Sprintf("Assertion errors in %s:", filename))
	for _, err := range errs {
		_, _ = color.New(color.BgBlack, color.FgRed).Println(err)
	}
	os.Exit(3)
}
