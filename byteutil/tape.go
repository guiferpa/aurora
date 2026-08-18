package byteutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/holiman/uint256"
)

// A tape is how every value in Aurora is represented: a fixed run of bytes, big-endian,
// right-aligned. How many bytes is a property of the program being compiled, not of the
// language, so it travels as a parameter instead of living here as a constant.
//
// This file is the *value* side of the package. The uint64 helpers (FromUint64, ToUint64,
// Padding64Bits) are the *metadata* side: instruction offsets, defer body lengths and
// argument indexes count instructions, not values, and stay 64-bit whatever the tape size.
const (
	DefaultTapeSize = 8
	MinTapeSize     = 1
	// MaxTapeSize is the EVM word: the backend cannot push a wider operand (PUSH32 is the
	// largest push) and the arithmetic carrier is 256 bits.
	MaxTapeSize = 32
)

// TapeSize normalizes a configured size. Zero means "unset", which is the common case for
// zero-valued option structs, and becomes the default; anything out of range is clamped.
//
// Clamping and refusing are two answers to the same question — what a tape can be — so they
// live side by side. A caller deciding whether to go on at all asks ValidateTapeSize; one that
// only needs a width to work with normalizes and carries on.
func TapeSize(size int) int {
	if size < MinTapeSize {
		return DefaultTapeSize
	}
	if size > MaxTapeSize {
		return MaxTapeSize
	}
	return size
}

// ValidateTapeSize refuses a size no tape can be, in the words whoever typed it has to read.
//
// Zero passes: it means the width was never set, and the default applies. What is refused is
// a width someone asked for and cannot have — silently clamping 64 to 32 would compile a
// program in a dialect nobody chose.
func ValidateTapeSize(size int) error {
	if size == 0 {
		return nil
	}
	if size < MinTapeSize || size > MaxTapeSize {
		return fmt.Errorf("tape size must be between %d and %d bytes, got %d",
			MinTapeSize, MaxTapeSize, size)
	}
	return nil
}

// ParseTapeSize reads a tape size written as text, answering with fallback when the text is
// not a number a tape can be.
//
// It exists for a host that takes the size from outside the program — a form control, a
// query string — where the value arrives as text and a bad one is not worth stopping for.
// The fallback is the caller's because a host may start from a width of its own: the
// playground opens at 32, not at the language's 8.
func ParseTapeSize(text string, fallback int) int {
	size, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || size < MinTapeSize || size > MaxTapeSize {
		return fallback
	}
	return size
}

// PaddingTape returns bs as a tape of exactly size bytes: zero-padded on the left when
// shorter, and cut to the last size bytes when longer — which is what keeps arithmetic on
// a run of tapes working on its last one.
func PaddingTape(bs []byte, size int) []byte {
	size = TapeSize(size)
	if len(bs) == size {
		return bs
	}
	if len(bs) > size {
		return bs[len(bs)-size:]
	}
	tape := make([]byte, size)
	copy(tape[size-len(bs):], bs)
	return tape
}

// LeadingTape returns the first size bytes of bs, zero-padding on the right when shorter.
// It is the mirror of PaddingTape, which keeps the last ones: a tape is a shift register,
// so an operation either keeps the left end or the right end and drops the rest.
func LeadingTape(bs []byte, size int) []byte {
	size = TapeSize(size)
	tape := make([]byte, size)
	copy(tape, bs)
	return tape
}

// ToUint256 reads bs as an unsigned big-endian integer, normalizing to a tape first so a
// run of tapes contributes only its last one.
func ToUint256(bs []byte, size int) *uint256.Int {
	return new(uint256.Int).SetBytes(PaddingTape(bs, size))
}

// FromUint256 writes v as a tape. Keeping the last size bytes is the wrap-around of the
// language: a tape of N bytes holds values modulo 2^(8N).
func FromUint256(v *uint256.Int, size int) []byte {
	size = TapeSize(size)
	full := v.Bytes32()
	tape := make([]byte, size)
	copy(tape, full[len(full)-size:])
	return tape
}

// TrueTape and FalseTape are the boolean values, and ordinary tapes: nothing
// distinguishes true from the number 1, which is the point of an untyped language.
//
// FalseTape is also the language's neutral value — what a scope with no value returns,
// what a binding evaluates to, the tape a literal starts from. There is no separate
// "nothing": a tape of zeros is a tape of zeros.
func TrueTape(size int) []byte {
	return PaddingTape([]byte{1}, size)
}

func FalseTape(size int) []byte {
	return make([]byte, TapeSize(size))
}

// FitsInTape reports whether v can be held by a tape of the given size. Used to reject a
// literal at compile time instead of truncating it silently.
func FitsInTape(v uint64, size int) bool {
	size = TapeSize(size)
	if size >= 8 {
		return true
	}
	return v < uint64(1)<<(8*size)
}

// MaxTapeValue is the largest value a tape of the given size can hold, for error messages.
func MaxTapeValue(size int) uint64 {
	size = TapeSize(size)
	if size >= 8 {
		return ^uint64(0)
	}
	return uint64(1)<<(8*size) - 1
}

// IsZeroTape reports whether every byte is zero.
func IsZeroTape(bs []byte) bool {
	for _, b := range bs {
		if b != 0 {
			return false
		}
	}
	return true
}
