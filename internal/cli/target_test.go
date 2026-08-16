package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const manifestWithProfiles = `[project]
  name = "example"
  version = "0.1.0"
  tape_size = 1

[profiles]
  [profiles.main]
    source = "src/main.ar"
    binary = "bin/main"

  [profiles.tiny]
    source = "src/tiny.ar"
    binary = "bin/tiny"
`

// projectDir builds a project with a manifest and makes it the working directory.
func projectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "aurora.toml"), []byte(manifestWithProfiles), 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	t.Chdir(dir)
	return dir
}

func TestResolveTargetFromProfile(t *testing.T) {
	dir := projectDir(t)

	cases := []struct {
		name       string
		arg        string
		wantSource string
		wantBinary string
		wantTape   int
		wantName   string
	}{
		// The width is the project's, so every profile of it compiles the same dialect.
		{name: "no argument means main", arg: "", wantSource: "src/main.ar", wantBinary: "bin/main", wantTape: 1, wantName: "main"},
		{name: "a name selects a profile", arg: "tiny", wantSource: "src/tiny.ar", wantBinary: "bin/tiny", wantTape: 1, wantName: "tiny"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveTarget(tc.arg)
			if err != nil {
				t.Fatalf("ResolveTarget(%q): %v", tc.arg, err)
			}
			if want := filepath.Join(dir, tc.wantSource); got.Source != want {
				t.Errorf("source = %q, want %q", got.Source, want)
			}
			if want := filepath.Join(dir, tc.wantBinary); got.Binary != want {
				t.Errorf("binary = %q, want %q", got.Binary, want)
			}
			if got.TapeSize != tc.wantTape {
				t.Errorf("tape size = %d, want %d", got.TapeSize, tc.wantTape)
			}
			if got.Profile != tc.wantName || !got.FromProfile() {
				t.Errorf("profile = %q, want %q", got.Profile, tc.wantName)
			}
		})
	}
}

// A path is taken as it is: it names no profile, so nothing is inherited from one. The
// width is the exception, and it is not the profile's — a file inside a project is written
// in that project's dialect however it was named.
func TestResolveTargetFromPath(t *testing.T) {
	projectDir(t)

	for _, arg := range []string{"main.ar", "src/main.ar", "../elsewhere/x.ar", "/tmp/abs.ar"} {
		t.Run(arg, func(t *testing.T) {
			got, err := ResolveTarget(arg)
			if err != nil {
				t.Fatalf("ResolveTarget(%q): %v", arg, err)
			}
			if got.Source != arg {
				t.Errorf("source = %q, want %q", got.Source, arg)
			}
			if got.FromProfile() {
				t.Error("a path must not resolve to a profile")
			}
			if got.Binary != "" {
				t.Errorf("a loose file carries no profile settings: %+v", got)
			}
		})
	}
}

// A file named by its path and the same file named by its profile have to compile the same
// way. They did not: the width was the profile's, so `aurora run src/main.ar` read the file
// at eight bytes while `aurora run` read it at the project's width.
func TestALooseFileTakesTheWidthOfTheProjectItIsIn(t *testing.T) {
	projectDir(t) // its manifest says tape_size = 1

	byPath, err := ResolveTarget(filepath.Join("src", "main.ar"))
	if err != nil {
		t.Fatalf("ResolveTarget by path: %v", err)
	}
	byProfile, err := ResolveTarget("")
	if err != nil {
		t.Fatalf("ResolveTarget by profile: %v", err)
	}

	if byPath.TapeSize != 1 {
		t.Errorf("by path the tape size is %d, want the project's 1", byPath.TapeSize)
	}
	if byPath.TapeSize != byProfile.TapeSize {
		t.Errorf("the same file reads %d bytes by path and %d by profile", byPath.TapeSize, byProfile.TapeSize)
	}
}

// A file outside any project keeps the default, which is what makes trying the language out
// cheap: no manifest, no project, still runs.
func TestALooseFileOutsideAProjectHasNoWidthOfItsOwn(t *testing.T) {
	t.Chdir(t.TempDir())

	got, err := ResolveTarget("program.ar")
	if err != nil {
		t.Fatalf("ResolveTarget: %v", err)
	}
	if got.TapeSize != 0 {
		t.Errorf("tape size = %d, want it unset so the default applies", got.TapeSize)
	}
}

// A manifest that cannot be read is said out loud rather than answered with the default: a
// project pinned to one byte compiling at eight in silence is what this guards against.
func TestALooseFileReportsABrokenManifest(t *testing.T) {
	dir := t.TempDir()
	const oldField = `[project]
name = "p"

[profiles.main]
source = "src/main.ar"
tape_size = 1
`
	if err := os.WriteFile(filepath.Join(dir, "aurora.toml"), []byte(oldField), 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	t.Chdir(dir)

	if _, err := ResolveTarget("src/main.ar"); err == nil {
		t.Error("the manifest was read and its refusal ignored")
	}
}

// The regression that motivated this: running a file must not require a project.
func TestResolveTargetFromPathNeedsNoManifest(t *testing.T) {
	t.Chdir(t.TempDir()) // no aurora.toml anywhere above

	got, err := ResolveTarget("program.ar")
	if err != nil {
		t.Fatalf("a path must not need a manifest: %v", err)
	}
	if got.Source != "program.ar" {
		t.Errorf("source = %q, want %q", got.Source, "program.ar")
	}
}

func TestResolveTargetErrors(t *testing.T) {
	t.Run("unknown profile", func(t *testing.T) {
		projectDir(t)
		if _, err := ResolveTarget("nope"); err == nil {
			t.Error("expected an error for a profile that is not in the manifest")
		}
	})

	t.Run("no manifest and no path", func(t *testing.T) {
		t.Chdir(t.TempDir())
		_, err := ResolveTarget("")
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "aurora.toml") {
			t.Errorf("error = %q, want it to name the missing manifest", err)
		}
	})

	// An argument that was meant as a file but lost its extension should say so, rather
	// than being looked up as a profile and failing for an unrelated reason.
	t.Run("path without the extension", func(t *testing.T) {
		projectDir(t)
		for _, arg := range []string{"./main", "src/main", "main.txt"} {
			_, err := ResolveTarget(arg)
			if err == nil {
				t.Errorf("%q: expected an error", arg)
				continue
			}
			if !strings.Contains(err.Error(), ".ar") {
				t.Errorf("%q: error = %q, want it to mention the .ar extension", arg, err)
			}
		}
	})
}

func TestDefaultBinaryPath(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: "main.ar", want: "main"},
		{source: "examples/arithmetic.ar", want: "arithmetic"},
		{source: "/abs/path/checks.test.ar", want: "checks.test"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			if got := DefaultBinaryPath(tc.source); got != tc.want {
				t.Errorf("DefaultBinaryPath(%q) = %q, want %q", tc.source, got, tc.want)
			}
		})
	}
}
