package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/stdlib"
)

// The standard library the binary carries, written where the language reads it from.
//
// The two are separate on purpose: what a program imports is read from disk, and this only
// puts it there. So the test is that what lands is exactly what a program will import, under
// the names it will import it by.
func TestInstallingTheStandardLibrary(t *testing.T) {
	root := t.TempDir()
	out := &strings.Builder{}

	if err := InstallStd(InstallStdInput{Files: stdlib.Files(), Root: root, Out: out}); err != nil {
		t.Fatalf("installing: %v", err)
	}

	// The path a module name resolves to, spelled the way the resolver spells it.
	installed := filepath.Join(root, "lib", "std", "evm", "storage.ar")
	source, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("reading what was installed: %v", err)
	}
	for _, wanted := range []string{"ident set = defer", "ident get = defer", "sstore", "sload"} {
		if !strings.Contains(string(source), wanted) {
			t.Errorf("what was installed has no %q in it", wanted)
		}
	}
	if !strings.Contains(out.String(), filepath.Join(root, "lib", "std")) {
		t.Errorf("it said %q, and never said where it went", out)
	}
}

// And a program reads what was installed, which is the only thing that makes it a standard
// library rather than a copied file.
func TestAProgramImportsWhatWasInstalled(t *testing.T) {
	root := t.TempDir()
	if err := InstallStd(InstallStdInput{Files: stdlib.Files(), Root: root, Out: &strings.Builder{}}); err != nil {
		t.Fatalf("installing: %v", err)
	}

	projectOf(t, map[string]string{
		"src/main.ar": "use std/evm/storage as s;\ns.set(1, 42);\nprintd s.get(1);",
	})
	printed, err := runWithStd(t, "src/main.ar", filepath.Join(root, "lib"))
	if err != nil {
		t.Fatalf("running: %v", err)
	}
	if got := strings.TrimSpace(printed); got != "42" {
		t.Errorf("printed %q, want 42", got)
	}
}

// One that is already there is left alone and said so, because somebody may have patched a
// module — and a command that silently undoes that is one nobody trusts twice.
func TestInstallingOverOneThatIsAlreadyThere(t *testing.T) {
	root := t.TempDir()
	installed := filepath.Join(root, "lib", "std", "evm", "storage.ar")
	if err := os.MkdirAll(filepath.Dir(installed), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed, []byte("#- patched by somebody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := InstallStd(InstallStdInput{Files: stdlib.Files(), Root: root, Out: &strings.Builder{}})
	if err == nil {
		t.Fatal("it wrote over a standard library somebody had already put there")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("it says %q, and never says how to do it anyway", err)
	}

	source, readErr := os.ReadFile(installed)
	if readErr != nil || !strings.Contains(string(source), "patched by somebody") {
		t.Error("it left something other than what was there")
	}

	// And with force, it is replaced.
	if err := InstallStd(InstallStdInput{Files: stdlib.Files(), Root: root, Force: true, Out: &strings.Builder{}}); err != nil {
		t.Fatalf("installing with force: %v", err)
	}
	source, err = os.ReadFile(installed)
	if err != nil || strings.Contains(string(source), "patched by somebody") {
		t.Error("force left what was there")
	}
}

// What the binary carries is what the repository has. A copy that drifts from the tree it came
// from is a standard library that answers differently depending on how somebody got it.
func TestWhatTheBinaryCarriesIsWhatTheRepositoryHas(t *testing.T) {
	carried, err := os.ReadFile(filepath.Join(repositoryLib(t), "std", "evm", "storage.ar"))
	if err != nil {
		t.Fatalf("reading the repository's own: %v", err)
	}

	root := t.TempDir()
	if err := InstallStd(InstallStdInput{Files: stdlib.Files(), Root: root, Out: &strings.Builder{}}); err != nil {
		t.Fatalf("installing: %v", err)
	}
	installed, err := os.ReadFile(filepath.Join(root, "lib", "std", "evm", "storage.ar"))
	if err != nil {
		t.Fatalf("reading what was installed: %v", err)
	}

	if string(carried) != string(installed) {
		t.Error("what the binary carries is not what the repository has")
	}
}
