package builtin

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/byteutil"
)

// PrintFunction writes the bytes of a value as they are: print is how you see what a value
// actually is.
func PrintFunction(w io.Writer, bs []byte) {
	_, _ = w.Write(bs)
}

// EchoFunction writes a value as text: echo is how you read bytes back as characters.
//
// A reel is a run of tapes, one character each, so anything wider than a tape is walked
// tape by tape. A single tape becomes the character its last significant byte stands for.
func EchoFunction(w io.Writer, bs []byte, tapeSize int) {
	_, _ = w.Write([]byte(textOf(bs, tapeSize)))
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
	return byteutil.PaddingTape(value, tapeSize)
}

// AssertFunction evaluates an assert: the condition as a truth value, the message as text
// to show when it does not hold.
func AssertFunction(cond, msg []byte, tapeSize int) (bool, error) {
	if byteutil.ToBoolean(cond) {
		return true, nil
	}
	return false, fmt.Errorf("assertion failed: %s", textOf(msg, tapeSize))
}

// textOf reads a value as text, tape by tape. Both echo and a failed assertion need it,
// and they used to walk the bytes in two slightly different ways — one of them with a
// stride of 8 that ignored the tape size, which read past the end of a narrow reel.
func textOf(bs []byte, tapeSize int) string {
	tapeSize = byteutil.TapeSize(tapeSize)
	if len(bs) == 0 {
		return ""
	}

	// A run of whole tapes is a reel: one character per tape.
	if len(bs) > tapeSize && len(bs)%tapeSize == 0 {
		text := make([]byte, 0, len(bs)/tapeSize)
		for i := 0; i < len(bs); i += tapeSize {
			if char, ok := printableOf(bs[i : i+tapeSize]); ok {
				text = append(text, char)
			}
		}
		return string(text)
	}

	// Anything else is a single value: the character its last significant byte stands for,
	// or the significant bytes themselves when that is not printable.
	if char, ok := printableOf(bs); ok {
		return string(rune(char))
	}
	return string(significantBytes(bs))
}

// printableOf returns the printable ASCII character a tape stands for, if it is one.
func printableOf(tape []byte) (byte, bool) {
	significant := significantBytes(tape)
	if len(significant) == 0 {
		return 0, false
	}
	char := significant[len(significant)-1]
	return char, char >= 32 && char <= 126
}

// significantBytes drops the leading zeros of a value, which are padding rather than
// content. A value of all zeros has nothing significant in it.
func significantBytes(bs []byte) []byte {
	for i := range bs {
		if bs[i] != 0 {
			return bs[i:]
		}
	}
	return nil
}
