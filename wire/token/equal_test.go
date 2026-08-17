package token

import "testing"

// The three tokens these cases are made of: the same text under two tags, and a different
// text, which is what tells the two halves of a comparison apart.
var (
	one        = New([]byte("1"), TagNumber, 1, 1, 0)
	two        = New([]byte("2"), TagNumber, 1, 1, 0)
	oneAsIdent = New([]byte("1"), TagIdent, 1, 1, 0)
)

// Equal reads both halves of a token, and nothing else about it.
func TestEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b Token
		want bool
	}{
		{name: "nothing against nothing", want: true},
		{name: "nothing against a token", b: one, want: false},
		{name: "a token against nothing", a: one, want: false},
		{name: "alike", a: one, b: New([]byte("1"), TagNumber, 0, 0, 0), want: true},
		{name: "another match", a: one, b: two, want: false},
		{name: "another tag", a: one, b: oneAsIdent, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Equal(tc.a, tc.b); got != tc.want {
				t.Errorf("answered %v, want %v", got, tc.want)
			}
		})
	}
}
