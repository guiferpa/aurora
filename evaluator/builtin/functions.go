package builtin

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/guiferpa/aurora/byteutil"
)

// Every value in Aurora is a tape of bytes, and the three print builtins are three ways of
// reading the same tape. Each says which one it is in its name:
//
//	printb   the bytes themselves       [0 0 0 0 0 0 0 44]
//	printd   as a decimal number        44
//	printc   as the character it names   ,
//
// A reel is a run of tapes, so each of them is read tape by tape.

// PrintBytesFunction writes the bytes of a value, which is what the value is.
func PrintBytesFunction(w io.Writer, bs []byte) {
	_, _ = fmt.Fprintf(w, "%v\n", bs)
}

// PrintDecimalFunction writes a value as a number, or one number per tape for a reel.
func PrintDecimalFunction(w io.Writer, bs []byte, tapeSize int) {
	_, _ = fmt.Fprintf(w, "%s\n", DecimalOf(bs, tapeSize))
}

// PrintCharsFunction writes a value as text.
func PrintCharsFunction(w io.Writer, bs []byte, tapeSize int) {
	_, _ = fmt.Fprintf(w, "%s\n", TextOf(bs, tapeSize))
}

// FeedFunction returns the value at the given index of the feed: the vector of values
// applied to the running scope. It is the builtin behind "feed(index)".
//
// Execution in Aurora is the application of a vector of values to a scope, not a function
// call — there is no signature, no arity and no parameter, so this only reads a position.
// The scope never learns how many values it got: the index wraps around the length of the
// vector, so the read always answers with a tape and never fails.
func FeedFunction(feed map[uint64][]byte, index uint64, tapeSize int) []byte {
	if len(feed) == 0 {
		return byteutil.FalseTape(tapeSize)
	}
	value, ok := feed[index%uint64(len(feed))]
	if !ok {
		return byteutil.FalseTape(tapeSize)
	}
	// A value wider than a tape is a run of them — a reel, or a struct — and it is handed
	// over whole. Narrowing here would cut a struct down to its last field, and the
	// narrowing that command-line arguments need already happens where they enter, in
	// environ.NewEnviron. Anything shorter than a tape is padded, so a read always answers
	// with at least one.
	if len(value) >= byteutil.TapeSize(tapeSize) {
		return value
	}
	return byteutil.PaddingTape(value, tapeSize)
}

// AssertFunction evaluates an assert: the condition as a truth value, the message as text
// to show when it does not hold.
func AssertFunction(cond, msg []byte, tapeSize int) (bool, error) {
	if byteutil.ToBoolean(cond) {
		return true, nil
	}
	return false, fmt.Errorf("assertion failed: %s", TextOf(msg, tapeSize))
}

// TextOf reads a value as text: each tape holds the number of a character, and that
// character is written as UTF-8.
//
// Reading the tape as a number rather than as a single byte is what lets a character
// outside ASCII survive — "café" holds 233 in its last tape, and 233 is é.
func TextOf(bs []byte, tapeSize int) string {
	var text strings.Builder
	for _, tape := range tapesOf(bs, tapeSize) {
		char := rune(byteutil.ToUint256(tape, tapeSize).Uint64())
		// A tape of zeros is the neutral value rather than a character, and a number that
		// names no character has nothing to write.
		if char == 0 || !utf8.ValidRune(char) {
			continue
		}
		text.WriteRune(char)
	}
	return text.String()
}

// DecimalOf reads a value as a number, or as one number per tape when it is a reel.
func DecimalOf(bs []byte, tapeSize int) string {
	tapes := tapesOf(bs, tapeSize)
	numbers := make([]string, 0, len(tapes))
	for _, tape := range tapes {
		numbers = append(numbers, byteutil.ToUint256(tape, tapeSize).Dec())
	}
	return strings.Join(numbers, " ")
}

// tapesOf splits a value into the tapes it holds. Anything that is not a whole run of
// tapes is read as a single one.
func tapesOf(bs []byte, tapeSize int) [][]byte {
	tapeSize = byteutil.TapeSize(tapeSize)
	if len(bs) == 0 {
		return nil
	}
	if len(bs) <= tapeSize || len(bs)%tapeSize != 0 {
		return [][]byte{bs}
	}

	tapes := make([][]byte, 0, len(bs)/tapeSize)
	for i := 0; i < len(bs); i += tapeSize {
		tapes = append(tapes, bs[i:i+tapeSize])
	}
	return tapes
}
