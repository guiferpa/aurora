package cli

import (
	"strings"
	"testing"
)

// The flag wins over the manifest, and the default applies when neither is set.
func TestResolveTapeSizePrecedence(t *testing.T) {
	cases := []struct {
		name     string
		flag     int
		manifest int
		want     int
	}{
		{name: "flag overrides the manifest", flag: 4, manifest: 2, want: 4},
		{name: "manifest applies when no flag", flag: 0, manifest: 2, want: 2},
		{name: "neither set leaves the default to the compiler", flag: 0, manifest: 0, want: 0},
		{name: "flag alone", flag: 1, manifest: 0, want: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveTapeSize(tc.flag, tc.manifest); got != tc.want {
				t.Errorf("ResolveTapeSize(%d, %d) = %d, want %d", tc.flag, tc.manifest, got, tc.want)
			}
		})
	}
}

func TestBuildRejectsInvalidTapeSize(t *testing.T) {
	_, err := newSession(t, sessionOpts{tapeSize: 64}).Build(t.Context(), "main.ar", "out.bin")
	if err == nil || !strings.Contains(err.Error(), "tape size") {
		t.Errorf("expected the tape size to be rejected before compiling, got %v", err)
	}
}
