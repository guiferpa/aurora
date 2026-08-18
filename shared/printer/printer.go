// Package printer writes what a program says.
//
// Every host has to do this and none of them should decide it twice: a printed value looks
// the same from the command line, from the REPL and from the page. The evaluator asks for it
// through a port and does not know any of this — where the words go is the host's, and what
// they look like is here.
package printer

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/byteutil"
)

// A Printer answers with the value of the expression that printed, which is the value it was
// given: a print reads a tape three ways and changes none of them.
//
// Answering with the formatted text instead would answer with something that is not a tape —
// a twenty-digit number read as decimal does not fit an eight-byte one — and every value in
// Aurora is a tape.
type Printer struct {
	out  io.Writer
	read func(value []byte, tapeSize int) string
	size int
}

func (p Printer) Print(value []byte) ([]byte, error) {
	if _, err := fmt.Fprintln(p.out, p.read(value, p.size)); err != nil {
		return nil, err
	}
	return value, nil
}

// Bytes writes the bytes a value holds, which is what the value is.
func Bytes(out io.Writer, tapeSize int) Printer {
	return Printer{out: out, size: tapeSize, read: func(value []byte, _ int) string {
		return fmt.Sprintf("%v", value)
	}}
}

// Chars writes a value as text.
func Chars(out io.Writer, tapeSize int) Printer {
	return Printer{out: out, size: tapeSize, read: byteutil.TextOf}
}

// Decimal writes a value as a number, one per tape.
func Decimal(out io.Writer, tapeSize int) Printer {
	return Printer{out: out, size: tapeSize, read: byteutil.DecimalOf}
}
