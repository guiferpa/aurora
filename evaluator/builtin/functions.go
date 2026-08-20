package builtin

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
)

// What a program can ask for by name: the values it was applied to, and an assertion about
// one of them. Both answer with a value, and neither touches the world.
//
// The prints used to live here as well. They wrote, which made this the one place where a
// vital phase reached for a writer; they are now ports the host fills in — see Printer in the
// evaluator, and how a tape is read in byteutil.

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
	// A value wider than a tape is a run of them — a reel, or a shape — and it is handed
	// over whole. Narrowing here would cut a shape down to its last field, and the
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
func AssertFunction(cond []byte, msg string) (bool, error) {
	if byteutil.ToBoolean(cond) {
		return true, nil
	}
	return false, fmt.Errorf("assertion failed: %s", msg)
}
