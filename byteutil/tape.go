package byteutil

import "github.com/holiman/uint256"

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
// Rejecting a bad size is the CLI's job, at the boundary where a good message can be given.
func TapeSize(size int) int {
	if size < MinTapeSize {
		return DefaultTapeSize
	}
	if size > MaxTapeSize {
		return MaxTapeSize
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
