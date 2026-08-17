package logger

import "github.com/fatih/color"

// How a failure is spelled, so every host spells it the same way.
//
// Nothing here decides where a message goes or what happens next. It used to: this wrote to
// stderr and ended the process, which is a package deciding the fate of the program that
// called it. Whoever calls now writes and exits, and this says only what the words are.

// CommandError spells a command that failed, and answers with nothing when none did.
func CommandError(err error) string {
	if err == nil {
		return ""
	}
	return color.New(color.BgBlack, color.FgHiMagenta).Sprintln(err)
}
