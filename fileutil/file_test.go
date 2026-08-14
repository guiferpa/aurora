package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func TestListFilesByExtension(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.ar")
	write(t, dir, "b.ar")
	write(t, dir, "c.txt")
	write(t, dir, "noext")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	write(t, filepath.Join(dir, "sub"), "deep.ar")

	got, err := ListFilesByExtension(dir, ".ar")
	if err != nil {
		t.Fatalf("ListFilesByExtension: %v", err)
	}

	want := []string{filepath.Join(dir, "a.ar"), filepath.Join(dir, "b.ar")}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// Only the first level is considered, so a directory whose name ends in the extension is
// not a match either.
func TestListFilesByExtensionSkipsDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "looks-like.ar"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := ListFilesByExtension(dir, ".ar")
	if err != nil {
		t.Fatalf("ListFilesByExtension: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

func TestListFilesByExtensionWithNoMatches(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.txt")

	got, err := ListFilesByExtension(dir, ".ar")
	if err != nil {
		t.Fatalf("ListFilesByExtension: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

func TestListFilesByExtensionOnMissingDirectory(t *testing.T) {
	if _, err := ListFilesByExtension(filepath.Join(t.TempDir(), "nope"), ".ar"); err == nil {
		t.Error("expected an error for a directory that does not exist")
	}
}
