package cli

import (
	"io"
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
	if !strings.Contains(content, "source = \"src/main.ar\"") {
		t.Errorf("manifest should contain source")
	}
	if !strings.Contains(content, "binary = \"bin/main\"") {
		t.Errorf("manifest should contain binary")
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

// A manifest on its own leaves someone with nothing to run, so init writes the layout it
// describes: the program the main profile points at, and its tests.
func TestInitWritesTheProjectLayout(t *testing.T) {
	dir := t.TempDir()
	if err := Init(InitInput{Dir: dir, ProjectName: "myproj"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	for _, name := range []string{"aurora.toml", "src/main.ar", "src/main.test.ar"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); err != nil {
			t.Errorf("%s was not created: %v", name, err)
		}
	}
}

// What init writes has to run and pass, or a new project starts broken.
func TestInitProjectRunsAndPasses(t *testing.T) {
	dir := t.TempDir()
	if err := Init(InitInput{Dir: dir, ProjectName: "myproj"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Chdir(dir)

	out := &strings.Builder{}
	if err := Run(t.Context(), RunInput{Source: filepath.Join("src", "main.ar"), Stdout: out}); err != nil {
		t.Fatalf("the program init wrote does not run: %v", err)
	}

	report, err := Test(t.Context(), TestInput{Stdout: io.Discard})
	if err != nil {
		t.Fatalf("the tests init wrote do not run: %v", err)
	}
	if !report.OK() {
		t.Errorf("the tests init wrote do not pass: %d failed", report.Failed)
	}
	if report.Passed == 0 {
		t.Error("expected assertions to have run")
	}
}

// The greeting is the point of the thing: Aurora is named after the author's daughter, and
// this is what she was saying at one year old. It is also what the test that comes with a
// new project checks.
func TestInitGreets(t *testing.T) {
	dir := t.TempDir()
	if err := Init(InitInput{Dir: dir}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Chdir(dir)

	out := &strings.Builder{}
	if err := Run(t.Context(), RunInput{
		Source: filepath.Join("src", "main.ar"),
		Stdout: out,
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "Abidu abide") {
		t.Errorf("a new project should say hello, got %q", out.String())
	}
}

// A file already sitting there belongs to whoever put it there.
func TestInitLeavesExistingSourcesAlone(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src", "main.ar")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(source, []byte("ident mine = 1;\n"), 0o644); err != nil {
		t.Fatalf("writing: %v", err)
	}

	if err := Init(InitInput{Dir: dir}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	bs, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if string(bs) != "ident mine = 1;\n" {
		t.Errorf("an existing file was overwritten:\n%s", bs)
	}
}

func TestInitReportsWhatItWrote(t *testing.T) {
	dir := t.TempDir()
	out := &strings.Builder{}

	if err := Init(InitInput{Dir: dir, Stdout: out}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	report := out.String()
	for _, want := range []string{"aurora.toml", "main.ar", "main.test.ar", "aurora run", "aurora test"} {
		if !strings.Contains(report, want) {
			t.Errorf("the report is missing %q:\n%s", want, report)
		}
	}
}
