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

[profiles]
  [profiles.main]
    source = "src/main.ar"
    binary = "bin/main"

  [profiles.tiny]
    source = "src/tiny.ar"
    binary = "bin/tiny"
    tape_size = 1
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
		{name: "no argument means main", arg: "", wantSource: "src/main.ar", wantBinary: "bin/main", wantName: "main"},
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

// A path is taken as it is: no manifest is read, so a loose file runs anywhere.
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
			if got.Binary != "" || got.TapeSize != 0 {
				t.Errorf("a loose file carries no profile settings: %+v", got)
			}
		})
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
