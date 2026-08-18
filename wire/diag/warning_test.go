package diag

import "testing"

// A warning is a message and sometimes a place. Whether it has one is the whole branch a host
// takes to write it, so it is what this answers for.

func TestAWarningSaysItsMessage(t *testing.T) {
	warning := Warning{Message: "if does not reach the bytecode yet", Line: 3, Column: 1}

	if got := warning.String(); got != "if does not reach the bytecode yet" {
		t.Errorf("said %q, want the message", got)
	}
}

// A backend answers about a program as a whole: the IR carries instructions, not lines, so
// there is no place to point at and a host must not write "main.ar:0:0".
func TestPositioned(t *testing.T) {
	cases := []struct {
		name    string
		warning Warning
		want    bool
	}{
		{
			name:    "a warning about a place in the source",
			warning: Warning{Message: "257 deferred scopes in one scope", Line: 12, Column: 5},
			want:    true,
		},
		{
			name:    "a warning about the program as a whole",
			warning: Warning{Message: "printd writes a log"},
			want:    false,
		},
		{
			// A column with no line is not a place either: the line is what points.
			name:    "a column and no line",
			warning: Warning{Message: "somewhere", Column: 5},
			want:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.warning.Positioned(); got != tc.want {
				t.Errorf("Positioned() = %v, want %v", got, tc.want)
			}
		})
	}
}
