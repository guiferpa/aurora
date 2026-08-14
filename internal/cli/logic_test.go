package cli

import "testing"

// `and` binds tighter than `or`, so `a and b or c` is `(a and b) or c`. The two used to share
// one level and group to the right, which read it as `a and (b or c)` and answered false for
// the first case here.
func TestLogicalPrecedence(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: "printd false and true or true;", want: "1"},
		{source: "printd true or false and false;", want: "1"},
		{source: "printd true or true and false;", want: "1"},
		{source: "printd false or true and true;", want: "1"},
		{source: "printd true and true or false;", want: "1"},
		{source: "printd false and false or false;", want: "0"},
		// A comparison binds tighter than either, which is what makes a range read.
		{source: "ident age = 20;\nprintd age bigger 18 and age smaller 65;", want: "1"},
		{source: "ident age = 70;\nprintd age bigger 18 and age smaller 65;", want: "0"},
	}

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			if got := output(t, tc.source, 0)[0]; got != tc.want {
				t.Errorf("%s printed %s, want %s", tc.source, got, tc.want)
			}
		})
	}
}

// Chains of one operator group to the left. Both operations are associative, so the answer
// is the same either way — this pins the grouping, not the result.
func TestLogicalChains(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: "printd true and true and true;", want: "1"},
		{source: "printd true and true and false;", want: "0"},
		{source: "printd false or false or true;", want: "1"},
		{source: "printd false or false or false;", want: "0"},
	}

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			if got := output(t, tc.source, 0)[0]; got != tc.want {
				t.Errorf("%s printed %s, want %s", tc.source, got, tc.want)
			}
		})
	}
}
