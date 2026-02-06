package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/guiferpa/aurora/manifest"
)

func TestEnviron_AbsPath(t *testing.T) {
	root := filepath.FromSlash("/project/root")
	env := &Environ{
		Root: root,
		Profile: manifest.Profile{
			Source: "src/main.ar",
			Binary: "bin/main",
		},
	}

	tests := []struct {
		path string
		want string
	}{
		{"src/main.ar", filepath.Join(root, "src", "main.ar")},
		{"bin/main", filepath.Join(root, "bin", "main")},
		{".aurora/key", filepath.Join(root, ".aurora", "key")},
	}
	for _, tt := range tests {
		got := env.AbsPath(tt.path)
		if got != tt.want {
			t.Errorf("AbsPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestEnviron_AbsPath_absoluteInput(t *testing.T) {
	env := &Environ{Root: "/project"}
	absPath := filepath.FromSlash("/absolute/path/to/file")
	got := env.AbsPath(absPath)
	if got != absPath {
		t.Errorf("AbsPath(absolute) should return as-is: got %q", got)
	}
}

func TestRequireManifest_failsWhenNoManifest(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	err := RequireManifest()
	if err == nil {
		t.Error("RequireManifest() expected error when no aurora.toml in tree")
	}
}

func TestRequireManifest_succeedsWhenManifestExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, manifest.Filename)
	if err := os.WriteFile(path, []byte("[project]\nname = \"x\"\nversion = \"0.1.0\"\n[profiles.main]\nsource = \"src/main.ar\"\nbinary = \"bin/main\""), 0o644); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	err := RequireManifest()
	if err != nil {
		t.Errorf("RequireManifest() with manifest present: %v", err)
	}
}
