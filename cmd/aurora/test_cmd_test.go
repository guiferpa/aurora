package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The command itself is thin — it reads flags and hands them to the handler — but the exit
// code it produces is a contract with a CI job, and that lives here.

func testProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	manifest := `[project]
  name = "demo"

[profiles]
  [profiles.main]
    source = "src/main.ar"
    binary = "bin/main"
`
	if err := os.WriteFile(filepath.Join(dir, "aurora.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Chdir(dir)
	return dir
}

func writeSource(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// runTestCmd drives the command the way the CLI does, with stdout redirected.
func runTestCmd(t *testing.T, args ...string) error {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = saved
		_ = r.Close()
	}()

	testCmd.SetArgs(args)
	runErr := testCmd.Execute()
	_ = w.Close()
	return runErr
}

func TestTestCommandPasses(t *testing.T) {
	dir := testProject(t)
	writeSource(t, filepath.Join(dir, "src"), "main.ar", "ident double = defer { feed(0) * 2; };\n")
	writeSource(t, filepath.Join(dir, "src"), "main.test.ar", `assert(double(4) equals 8, "doubles");`)

	if err := runTestCmd(t); err != nil {
		t.Errorf("a passing suite should not error: %v", err)
	}
}

// A failure has to reach the exit code, or a CI job would treat a broken suite as green.
func TestTestCommandFails(t *testing.T) {
	dir := testProject(t)
	writeSource(t, filepath.Join(dir, "src"), "main.ar", "ident a = 1;\n")
	writeSource(t, filepath.Join(dir, "src"), "main.test.ar", `assert(a equals 2, "does not hold");`)

	err := runTestCmd(t)
	if err == nil {
		t.Fatal("a failing assertion should surface as an error")
	}
	if !strings.Contains(err.Error(), "assertions failed") {
		t.Errorf("error = %q, want it to count the failures", err)
	}
}

func TestTestCommandWithAFileThatCannotRun(t *testing.T) {
	dir := testProject(t)
	writeSource(t, filepath.Join(dir, "src"), "main.ar", "ident a = 1;\n")
	writeSource(t, filepath.Join(dir, "src"), "main.test.ar", `assert(a equals 1, "holds");`)
	writeSource(t, filepath.Join(dir, "src"), "orphan.test.ar", `assert(1 equals 1, "no source");`)

	err := runTestCmd(t)
	if err == nil {
		t.Fatal("a test without its source should surface as an error")
	}
	if !strings.Contains(err.Error(), "could not run") {
		t.Errorf("error = %q, want it to say a file could not run", err)
	}
}

func TestTestCommandWithATapeSize(t *testing.T) {
	dir := testProject(t)
	writeSource(t, filepath.Join(dir, "src"), "main.ar", "ident a = 200;\n")
	writeSource(t, filepath.Join(dir, "src"), "main.test.ar", `assert(a + 100 equals 44, "wraps on one byte");`)

	if err := runTestCmd(t, "--tape-size", "1"); err != nil {
		t.Errorf("the flag should have set a one-byte tape: %v", err)
	}
}

func TestTestCommandWithAPath(t *testing.T) {
	dir := testProject(t)
	writeSource(t, filepath.Join(dir, "src"), "main.ar", "ident a = 1;\n")
	writeSource(t, filepath.Join(dir, "src"), "main.test.ar", `assert(a equals 1, "holds");`)

	if err := runTestCmd(t, filepath.Join("src", "main.test.ar")); err != nil {
		t.Errorf("naming a file should run it: %v", err)
	}
}

func TestTestCommandWithNothingToRun(t *testing.T) {
	dir := testProject(t)
	writeSource(t, filepath.Join(dir, "src"), "main.ar", "ident a = 1;\n")

	err := runTestCmd(t)
	if err == nil || !strings.Contains(err.Error(), "no .test.ar") {
		t.Errorf("error = %v, want it to say there is nothing to run", err)
	}
}
