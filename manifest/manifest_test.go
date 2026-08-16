package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write puts a manifest in a directory of its own and answers with the directory.
func write(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(contents), 0o644); err != nil {
		t.Fatalf("writing the manifest: %v", err)
	}
	return dir
}

// The width of a value is the project's, so it is read from [project] and nowhere else.
func TestLoadReadsTheProjectTapeSize(t *testing.T) {
	dir := write(t, `
[project]
name = "p"
tape_size = 16

[profiles.main]
source = "src/main.ar"
`)

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if m.Project.TapeSize != 16 {
		t.Errorf("tape size is %d, want 16", m.Project.TapeSize)
	}
}

// Unset means the default applies, and the default is decided where values are made, not
// here: the manifest says nothing rather than saying eight.
func TestLoadLeavesTheTapeSizeUnsetWhenTheProjectDoesNotSayIt(t *testing.T) {
	dir := write(t, `
[project]
name = "p"

[profiles.main]
source = "src/main.ar"
`)

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if m.Project.TapeSize != 0 {
		t.Errorf("tape size is %d, want it unset", m.Project.TapeSize)
	}
}

// A manifest still carrying the old field is refused. Reading it and doing nothing would
// leave a project pinned to one byte compiling at eight without a word.
func TestLoadRefusesATapeSizeInsideAProfile(t *testing.T) {
	dir := write(t, `
[project]
name = "p"

[profiles.tiny]
source = "src/tiny.ar"
tape_size = 1
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("the manifest was accepted")
	}
	for _, want := range []string{"tape_size", "profiles.tiny", "[project]"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error says %q, want it to name %q", err, want)
		}
	}
}

// A key nobody reads is not an error by itself: only the one that moved is.
func TestLoadAcceptsAKeyItDoesNotKnow(t *testing.T) {
	dir := write(t, `
[project]
name = "p"

[profiles.main]
source = "src/main.ar"
something_else = true
`)

	if _, err := Load(dir); err != nil {
		t.Errorf("loading: %v", err)
	}
}

// The project a file belongs to is decided by where the file is.
func TestFindProjectRootFrom(t *testing.T) {
	root := write(t, "[project]\nname = \"p\"\n")
	deep := filepath.Join(root, "src", "nested")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("making the directories: %v", err)
	}

	got, err := FindProjectRootFrom(deep)
	if err != nil {
		t.Fatalf("walking up: %v", err)
	}
	// The temporary directory can be reached by more than one path, so compare what the
	// filesystem resolves them to.
	want, _ := filepath.EvalSymlinks(root)
	if resolved, _ := filepath.EvalSymlinks(got); resolved != want {
		t.Errorf("found %q, want %q", got, want)
	}
}

func TestFindProjectRootFromSaysWhenThereIsNoProject(t *testing.T) {
	_, err := FindProjectRootFrom(t.TempDir())
	if err == nil {
		t.Fatal("a directory with no manifest was taken for a project")
	}
	if !strings.Contains(err.Error(), Filename) {
		t.Errorf("the error says %q, want it to name %s", err, Filename)
	}
}
