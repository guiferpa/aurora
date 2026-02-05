package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/manifest"
)

func TestInit_createsManifest(t *testing.T) {
	dir := t.TempDir()
	err := Init(InitInput{Dir: dir, ProjectName: "myproj"})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	path := filepath.Join(dir, manifest.Filename)
	bs, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	content := string(bs)
	if !strings.Contains(content, `name = "myproj"`) {
		t.Errorf("manifest should contain name = \"myproj\", got:\n%s", content)
	}
	if !strings.Contains(content, "entrypoint = \"src/main.ar\"") {
		t.Errorf("manifest should contain entrypoint")
	}
	if !strings.Contains(content, "target = \"dist/main\"") {
		t.Errorf("manifest should contain target")
	}
}

func TestInit_failsWhenDirEmpty(t *testing.T) {
	err := Init(InitInput{Dir: ""})
	if err == nil {
		t.Error("Init() with empty Dir should return error")
	}
	if !strings.Contains(err.Error(), "Dir is required") {
		t.Errorf("error should mention Dir: %v", err)
	}
}

func TestInit_failsWhenManifestExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, manifest.Filename)
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Init(InitInput{Dir: dir, ProjectName: "x"})
	if err == nil {
		t.Error("Init() when aurora.toml exists should return error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention already exists: %v", err)
	}
}

func TestInit_usesDirBaseWhenProjectNameEmpty(t *testing.T) {
	dir := t.TempDir()
	// Create a subdir so Base is predictable
	sub := filepath.Join(dir, "myapp")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	err := Init(InitInput{Dir: sub}) // ProjectName empty
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	path := filepath.Join(sub, manifest.Filename)
	bs, _ := os.ReadFile(path)
	if !strings.Contains(string(bs), `name = "myapp"`) {
		t.Errorf("manifest name should be dir base \"myapp\", got:\n%s", string(bs))
	}
}
