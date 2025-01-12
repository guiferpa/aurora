package logger

import (
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
