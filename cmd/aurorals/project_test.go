package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The width of a value is the project's, and how the server finds it is the server's own
// business: it walks up from the file, the way the CLI does. What it finds is handed to the
// analysis, which knows nothing about manifests.

// project writes a manifest and returns the path of a source file beside it.
func project(t *testing.T, manifest string) string {
	t.Helper()
	dir := t.TempDir()
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(dir, "aurora.toml"), []byte(manifest), 0o644); err != nil {
			t.Fatalf("writing the manifest: %v", err)
		}
	}
	return filepath.Join(dir, "main.ar")
}

func TestTapeSizeForReadsTheProject(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		want     int
	}{
		{
			name:     "a project that says how wide a value is",
			manifest: "[project]\nname = \"p\"\ntape_size = 16\n",
			want:     16,
		},
		{
			// Unset is answered as unset: what the default is belongs to the compiler.
			name:     "a project that says nothing",
			manifest: "[project]\nname = \"p\"\n",
			want:     0,
		},
		{
			name: "a file with no project above it",
			want: 0,
		},
		{
			// A project's mistake is the CLI's to report out loud; the editor keeps working.
			name:     "a manifest that cannot be read",
			manifest: "[project\nname = ",
			want:     0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tapeSizeFor(project(t, tc.manifest)); got != tc.want {
				t.Errorf("tape size = %d, want %d", got, tc.want)
			}
		})
	}
}

// A manifest edited while the editor is open has to reach the next keystroke: the answer is
// cached per directory, and the cache is only good while the manifest is the one that was
// read.
func TestTapeSizeForFollowsTheManifestBeingEdited(t *testing.T) {
	source := project(t, "[project]\nname = \"p\"\ntape_size = 1\n")
	path := filepath.Join(filepath.Dir(source), "aurora.toml")

	if got := tapeSizeFor(source); got != 1 {
		t.Fatalf("tape size = %d, want 1", got)
	}

	if err := os.WriteFile(path, []byte("[project]\nname = \"p\"\ntape_size = 16\n"), 0o644); err != nil {
		t.Fatalf("rewriting the manifest: %v", err)
	}
	// Modification times have a resolution of their own, and a rewrite inside the same tick
	// would read as unchanged.
	now := timeAfter(t, path)
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatalf("touching the manifest: %v", err)
	}

	if got := tapeSizeFor(source); got != 16 {
		t.Errorf("tape size = %d after the project was widened, want 16", got)
	}
}

// timeAfter returns a modification time later than the file's, so a rewrite is visible
// however coarse the filesystem's clock is.
func timeAfter(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("reading the manifest: %v", err)
	}
	return info.ModTime().Add(time.Second)
}

// A manifest that goes away leaves the file with no project, rather than with the width it
// had while it was there.
func TestTapeSizeForNoticesAManifestRemoved(t *testing.T) {
	source := project(t, "[project]\nname = \"p\"\ntape_size = 16\n")

	if got := tapeSizeFor(source); got != 16 {
		t.Fatalf("tape size = %d, want 16", got)
	}
	if err := os.Remove(filepath.Join(filepath.Dir(source), "aurora.toml")); err != nil {
		t.Fatalf("removing the manifest: %v", err)
	}

	if got := tapeSizeFor(source); got != 0 {
		t.Errorf("tape size = %d with no manifest left, want it unset", got)
	}
}
