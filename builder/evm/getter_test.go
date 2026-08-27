package evm

import "testing"

// Where each value applied to a scope sits in the calldata: the selector first, then a word
// each.
//
// It answers a number and not a byte, and that is the whole of what was wrong with it. The
// eighth sits at 256, and 256 in a byte is zero — so the eighth value read the selector and the
// ninth read the first, and the contract answered something rather than failing.
func TestWhereEachValueAppliedSits(t *testing.T) {
	for _, tc := range []struct {
		name  string
		nth   uint64
		where int
	}{
		{name: "the first, past the selector", nth: 0, where: 0x20},
		{name: "the second", nth: 1, where: 0x40},
		{name: "the third", nth: 2, where: 0x60},
		// The one that used to come out as zero.
		{name: "the eighth, past what a byte holds", nth: 7, where: 256},
		{name: "the ninth", nth: 8, where: 288},
		{name: "the thirty-second", nth: 31, where: 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := GetCalldataArgsOffset(tc.nth); got != tc.where {
				t.Errorf("value %d sits at %d, want %d", tc.nth, got, tc.where)
			}
		})
	}
}
