package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/guiferpa/aurora/hosting/lsp/state"
)

// The editor's buffer is what counts, not the file on disk.
//
// Somebody editing a module has not saved it, and a server answering from the disk would be
// answering about a version they have already moved past: a name they just wrote would be
// underlined as missing, and one they just deleted would go on resolving.
func TestReadingPrefersTheOpenBuffer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "geometry.ar")
	if err := os.WriteFile(path, []byte("ident onDisk = 1;"), 0o644); err != nil {
		t.Fatal(err)
	}

	documents := state.New()
	read := readThroughBuffers(documents)

	if got, err := read(path); err != nil || string(got) != "ident onDisk = 1;" {
		t.Fatalf("read %q (%v), want what is on disk", got, err)
	}

	documents.UpdateDocument("file://"+path, "ident inTheEditor = 2;")
	if got, err := read(path); err != nil || string(got) != "ident inTheEditor = 2;" {
		t.Errorf("read %q (%v), want what the editor is showing", got, err)
	}
}

// A file nobody has open is read from the disk, and one that is nowhere says so.
func TestReadingFallsBackToTheDisk(t *testing.T) {
	read := readThroughBuffers(state.New())

	if _, err := read(filepath.Join(t.TempDir(), "gone.ar")); err == nil {
		t.Error("a file that is nowhere was read")
	}
}

// Where a module name resolves from is the project's, and src/ next to the file when there is
// no project. It is absolute either way: a server has no working directory worth speaking of.
func TestSourceRootFor(t *testing.T) {
	dir := t.TempDir()
	loose := filepath.Join(dir, "loose.ar")

	if got, want := sourceRootFor(loose), filepath.Join(dir, "src"); got != want {
		t.Errorf("source root is %q, want %q", got, want)
	}

	manifest := "[project]\nname = \"p\"\nsource_root = \"lib\"\n"
	if err := os.WriteFile(filepath.Join(dir, "aurora.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, want := sourceRootFor(loose), filepath.Join(dir, "lib"); got != want {
		t.Errorf("source root is %q, want %q", got, want)
	}
}
