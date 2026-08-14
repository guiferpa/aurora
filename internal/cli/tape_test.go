package cli

import (
	"strings"
	"testing"
)

func TestValidateTapeSize(t *testing.T) {
	cases := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{name: "unset is fine", size: 0},
		{name: "floor", size: 1},
		{name: "default", size: 8},
		{name: "ceiling", size: 32},
		{name: "below the floor", size: -1, wantErr: true},
		{name: "above the ceiling", size: 33, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTapeSize(tc.size)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				if !strings.Contains(err.Error(), "between 1 and 32") {
					t.Errorf("error = %q, want it to state the range", err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

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
	_, err := Build(t.Context(), BuildInput{Source: "main.ar", OutputPath: "out.bin", TapeSize: 64})
	if err == nil || !strings.Contains(err.Error(), "tape size") {
		t.Errorf("expected the tape size to be rejected before compiling, got %v", err)
	}
}
