package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

// Where the toolchain's own files live: what somebody chose, or one place under the home
// directory when nobody chose.
//
// There is no third answer and no search. A list of places to look is how a machine ends up
// running the standard library of a toolchain that was uninstalled a year ago, and the point
// of one directory is that the version question has one answer.
func TestWhereTheToolchainKeepsItsOwnFiles(t *testing.T) {
	t.Run("what somebody chose", func(t *testing.T) {
		chosen := t.TempDir()
		t.Setenv(AuroraRootVariable, chosen)

		if got := AuroraRoot(); got != chosen {
			t.Errorf("the root is %q, want %q", got, chosen)
		}
		if got, want := StdRoot(), filepath.Join(chosen, "lib"); got != want {
			t.Errorf("the modules read from %q, want %q", got, want)
		}
	})

	t.Run("and where it looks when nobody did", func(t *testing.T) {
		t.Setenv(AuroraRootVariable, "")

		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("this machine has no home directory to look under")
		}
		if got, want := AuroraRoot(), filepath.Join(home, ".aurora"); got != want {
			t.Errorf("the root is %q, want %q", got, want)
		}
	})
}

// The prefix stays in the path rather than being stripped.
//
// It is part of the module's name: lib/std/evm/storage.ar and lib/evm/storage.ar would be two
// files, and only the first of them is what `use std/evm/storage` means.
func TestTheStdPrefixIsPartOfThePath(t *testing.T) {
	chosen := t.TempDir()
	t.Setenv(AuroraRootVariable, chosen)

	if got, want := StdRoot(), filepath.Join(chosen, "lib"); got != want {
		t.Errorf("the modules read from %q, want %q — the std stays in the module name", got, want)
	}
}
