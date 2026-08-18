package byteutil

import (
	"strings"
	"unicode/utf8"
)

// The three readings of a tape.
//
// A tape is bytes, and how those bytes are read — as the bytes themselves, as text, as a
// number — is a question about the tape rather than about whoever is showing it. They used to
// live beside the print builtins, which meant anything that wanted to read a tape had to
// reach into the evaluator.

// TextOf reads a value as text: the bytes it holds, as UTF-8.
//
// A tape is bytes, and text is those bytes — "café" is its five UTF-8 bytes, not four
// character numbers. Zeros are dropped rather than written: they are the padding that fills
// a tape out to its width, no text can contain one, and a UTF-8 sequence never does either.
// That is also what lets a run of tapes each holding a character still read as one word.
func TextOf(bs []byte, tapeSize int) string {
	text := make([]byte, 0, len(bs))
	for _, b := range bs {
		if b == 0 {
			continue
		}
		text = append(text, b)
	}
	if !utf8.Valid(text) {
		// Bytes that are not text have nothing to write; the value is still whatever it is.
		return ""
	}
	return string(text)
}

// DecimalOf reads a value as a number, or as one number per tape when it is a reel.
func DecimalOf(bs []byte, tapeSize int) string {
	tapes := tapesOf(bs, tapeSize)
	numbers := make([]string, 0, len(tapes))
	for _, tape := range tapes {
		numbers = append(numbers, ToUint256(tape, tapeSize).Dec())
	}
	return strings.Join(numbers, " ")
}

// tapesOf splits a value into the tapes it holds. Anything that is not a whole run of
// tapes is read as a single one.
func tapesOf(bs []byte, tapeSize int) [][]byte {
	tapeSize = TapeSize(tapeSize)
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
