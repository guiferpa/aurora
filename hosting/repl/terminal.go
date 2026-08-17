package repl

import (
	"os"

	"golang.org/x/term"
)

// This file is the only place in the project that knows about golang.org/x/term.
// The editor and the history work over plain io.Reader/io.Writer, so replacing the
// terminal backend (or dropping the dependency) means rewriting this file alone.

// isTTY reports whether f is an interactive terminal. Pipes and files are not,
// and for those the REPL falls back to plain line scanning.
func isTTY(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// enterRaw puts f in raw mode (no line buffering, no echo, no signal generation) and
// returns a function that restores the previous state. The REPL holds raw mode only
// while reading a line, so evaluation output keeps working with plain "\n".
func enterRaw(f *os.File) (func(), error) {
	fd := int(f.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() { _ = term.Restore(fd, state) }, nil
}
