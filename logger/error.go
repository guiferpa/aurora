package logger

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

func MustError[V any](v V, err error) V {
	if err == nil {
		return v
	}
	color.New(color.BgBlack, color.FgHiRed).Println(err)
	os.Exit(1)
	return v
}

func CommandError(err error) {
	if err == nil {
		return
	}
	color.New(color.BgBlack, color.FgHiMagenta).Println(err)
	os.Exit(2)
}

func AssertError(errs []string, filename string) {
	if len(errs) == 0 {
		return
	}
	color.New(color.FgWhite).Println(fmt.Sprintf("Assertion errors in %s:", filename))
	for _, err := range errs {
		color.New(color.BgBlack, color.FgRed).Println(err)
	}
	os.Exit(3)
}
