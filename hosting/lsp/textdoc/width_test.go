package textdoc

import (
	"strings"
	"testing"
)

// How wide a value is decides what fits in one, and the parser refuses what does not: a
// number too large for a tape, text longer than one, a tape literal with more items than one
// holds. The width arrives with the document — where it came from is the host's business —
// and this is what proves it is used rather than carried.

// "Guilherme" is nine bytes: it does not fit the default tape and does fit a sixteen-byte
// one. Reading a document at the wrong width is how the editor came to underline this while
// the compiler took it without a word.
const nineBytes = `printc "Guilherme";`

func TestTheWidthOfADocumentReachesTheParser(t *testing.T) {
	cases := []struct {
		name    string
		width   int
		wantErr bool
	}{
		{name: "wide enough", width: 16},
		{name: "the default, which is not", width: 0, wantErr: true},
		{name: "the default said out loud", width: 8, wantErr: true},
		{name: "narrower than the default", width: 1, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics := session().ValidateCode(Document{
				Filename: "main.ar",
				Source:   nineBytes,
				TapeSize: tc.width,
			})

			if tc.wantErr && len(diagnostics) == 0 {
				t.Error("the text was accepted, want it reported as too long for a tape")
			}
			if !tc.wantErr && len(diagnostics) != 0 {
				t.Errorf("the text was reported: %v", diagnostics[0].Message)
			}
		})
	}
}

// A number too large for the tape is the other half of the same rule, and the message names
// the width it was read at.
func TestANumberIsCheckedAgainstTheDocumentsWidth(t *testing.T) {
	diagnostics := session().ValidateCode(Document{Filename: "main.ar", Source: "printd 300;", TapeSize: 1})

	if len(diagnostics) == 0 {
		t.Fatal("300 was accepted on a one-byte tape")
	}
	if !strings.Contains(diagnostics[0].Message, "1-byte tape") {
		t.Errorf("the diagnostic says %q, want it to name the width", diagnostics[0].Message)
	}
}
