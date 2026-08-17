package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot answers with the directory holding go.mod, found by walking up from wherever the
// test happens to run.
//
// The tests that read docs/ and examples/ used to count directories — "..", ".." — which is
// a fact about where this package sits rather than about what it is reading. Moving the
// package out of internal/ broke all five at once, and they would have broken again on the
// next move.
func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("looking for the repository root: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the working directory: the repository root is not there")
		}
		dir = parent
	}
}
