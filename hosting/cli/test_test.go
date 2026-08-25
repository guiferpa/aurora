package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// project builds a project with a manifest whose main profile points at src/main.ar, and
// makes it the working directory.
func project(t *testing.T) string {
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

func writeAt(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return path
}

// A test reaches the code it checks by naming it, the way every other file does.
func TestATestImportsWhatItChecks(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	writeAt(t, dir, "src/geometry.ar", "ident double = defer { feed(0) * 2; };\nident base = 10;\n")
	writeAt(t, dir, "src/geometry.test.ar", `use geometry as g;
assert(g.double(4) equals 8, "doubles");
assert(g.base equals 10, "sees what it binds too");
`)

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if file := report.Files[0]; file.Err != nil {
		t.Fatalf("the test did not run: %v", file.Err)
	}
	if report.Passed != 2 || report.Failed != 0 {
		t.Errorf("got %d passed and %d failed, want 2 and 0", report.Passed, report.Failed)
	}
}

// A test file no longer belongs to a source file of the same name, so there does not have to
// be one. It is a program like any other, and a program that imports nothing is a program.
func TestATestNeedsNoSourceOfItsOwnName(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	writeAt(t, dir, "src/alone.test.ar", `assert(2 * 21 equals 42, "checks what it says itself");`)

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if !report.OK() {
		t.Errorf("a test with no source next to it should run: %+v", report.Files)
	}
}

// Two modules are two scopes, so a test may bind a name the module it checks also binds.
// Under the old rule the two files were one scope and this was a conflict.
func TestATestMayBindANameItsModuleAlsoBinds(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	writeAt(t, dir, "src/counter.ar", "ident total = 1;\n")
	writeAt(t, dir, "src/counter.test.ar", `use counter as c;
ident total = 2;
assert(total equals 2, "its own");
assert(c.total equals 1, "and the module's");
`)

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if file := report.Files[0]; file.Err != nil {
		t.Fatalf("the two names collided: %v", file.Err)
	}
	if report.Failed != 0 {
		t.Errorf("got %d failures: %+v", report.Failed, report.Files[0].Results)
	}
}

// The search starts at the directory of the profile's source and goes down. Anything above
// it belongs to another part of the project.
func TestSearchesFromTheSourceDirectoryDown(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(1 equals 1, "root of the search");`)
	writeAt(t, dir, "src/utils/text.test.ar", `assert(2 equals 2, "a leaf");`)
	// Above the starting point, so out of reach.
	writeAt(t, dir, "outside.test.ar", `assert(3 equals 4, "must not run");`)

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}

	if len(report.Files) != 2 {
		t.Fatalf("ran %d files, want 2: %v", len(report.Files), report.Files)
	}
	for _, file := range report.Files {
		if strings.Contains(file.Path, "outside") {
			t.Errorf("%s is above the source directory and should not have run", file.Path)
		}
	}
	if report.Failed != 0 {
		t.Errorf("got %d failures, want none", report.Failed)
	}
}

func TestRunsASingleFileByPath(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(1 equals 1, "one");`)
	writeAt(t, dir, "src/other.test.ar", `assert(2 equals 2, "two");`)

	report, err := tested(t, filepath.Join("src", "other.test.ar"), sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if len(report.Files) != 1 {
		t.Fatalf("ran %d files, want 1", len(report.Files))
	}
	if !strings.Contains(report.Files[0].Path, "other") {
		t.Errorf("ran %q, want the file that was named", report.Files[0].Path)
	}
}

func TestReportsFailures(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	writeAt(t, dir, "src/main.test.ar", `ident a = 1;
assert(a equals 1, "holds");
assert(a equals 2, "does not hold");
assert(a bigger 0, "still runs after a failure");
`)

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}

	if report.Passed != 2 || report.Failed != 1 {
		t.Errorf("got %d passed and %d failed, want 2 and 1", report.Passed, report.Failed)
	}
	if report.OK() {
		t.Error("a failed assertion must not report OK")
	}
	if report.Files[0].Passed() {
		t.Error("the file holds a failure")
	}

	// A failure does not stop the ones after it.
	results := report.Files[0].Results
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	if !results[2].Passed {
		t.Error("the assertion after the failure should still have run")
	}
}

// A shape crosses from the module a test names, which is how a test builds one: the shape
// and the name both travel, and a field is an index resolved while parsing.
func TestAShapeCrossesFromTheModuleATestNames(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	writeAt(t, dir, "src/point.ar", "shape Point { x, y };\nident origin = Point{0, 0};\n")
	writeAt(t, dir, "src/point.test.ar", `use point as p;
ident here = p.Point{10, 20};
assert(here.y equals 20, "a shape crosses");
assert((p.origin as p.Point).x equals 0, "and a value claimed with as reads too");
`)

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if file := report.Files[0]; file.Err != nil {
		t.Fatalf("the test did not compile: %v", file.Err)
	}
	if !report.OK() {
		t.Errorf("an assertion did not hold: %+v", report.Files[0].Results)
	}
}

// A module is compiled without its test, so it cannot lean on what the test declares — under
// `aurora run` there is no test at all.
func TestAModuleDoesNotSeeWhatItsTestDeclares(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	writeAt(t, dir, "src/point.ar", "ident p = Point{1, 2};\n")
	writeAt(t, dir, "src/point.test.ar", "use point as p;\nshape Point { x, y };\nassert(1 equals 1, \"unreached\");\n")

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if file := report.Files[0]; file.Err == nil {
		t.Fatal("the module built a shape only its test declares")
	}
}

func TestUsesTheProjectTapeSize(t *testing.T) {
	dir := t.TempDir()
	manifest := `[project]
  name = "demo"
  tape_size = 1

[profiles]
  [profiles.main]
    source = "src/main.ar"
    binary = "bin/main"
`
	if err := os.WriteFile(filepath.Join(dir, "aurora.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	t.Chdir(dir)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	// 200 + 100 wraps to 44 on a one-byte tape.
	writeAt(t, dir, "src/main.test.ar", `assert(200 + 100 equals 44, "wraps at the tape width");`)

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if report.Failed != 0 {
		t.Errorf("the project's tape size should have applied: %+v", report.Files[0].Results)
	}
}

func TestFlagOverridesTheProfileTapeSize(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(200 + 100 equals 44, "wraps only on one byte");`)

	report, err := tested(t, "", sessionOpts{stdout: io.Discard, tapeSize: 1})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if report.Failed != 0 {
		t.Error("the flag should have set a one-byte tape")
	}
}

func TestErrors(t *testing.T) {
	t.Run("no test files", func(t *testing.T) {
		dir := project(t)
		writeAt(t, dir, "src/main.ar", "ident a = 1;\n")

		_, err := tested(t, "", sessionOpts{stdout: io.Discard})
		if err == nil || !strings.Contains(err.Error(), "no .test.ar files") {
			t.Errorf("error = %v, want it to say there is nothing to run", err)
		}
	})

	t.Run("unknown profile", func(t *testing.T) {
		project(t)
		if _, err := tested(t, "nope", sessionOpts{stdout: io.Discard}); err == nil {
			t.Error("expected an error for a profile that is not in the manifest")
		}
	})

	t.Run("invalid tape size", func(t *testing.T) {
		project(t)
		_, err := tested(t, "", sessionOpts{stdout: io.Discard, tapeSize: 64})
		if err == nil || !strings.Contains(err.Error(), "tape size") {
			t.Errorf("error = %v, want the tape size to be rejected", err)
		}
	})

	t.Run("a test that does not compile", func(t *testing.T) {
		dir := project(t)
		writeAt(t, dir, "src/main.ar", "printd 1;\n")
		writeAt(t, dir, "src/main.test.ar", "assert(@@@);\n")

		report, err := tested(t, "", sessionOpts{stdout: io.Discard})
		if err != nil {
			t.Fatalf("Test: %v", err)
		}
		if report.OK() {
			t.Error("a file that does not compile is a failure")
		}
	})

	t.Run("a module the test names does not compile", func(t *testing.T) {
		dir := project(t)
		writeAt(t, dir, "src/main.ar", "printd 1;\n")
		writeAt(t, dir, "src/broken.ar", "ident = ;\n")
		writeAt(t, dir, "src/broken.test.ar", "use broken as b;\nassert(1 equals 1, \"never runs\");\n")

		report, err := tested(t, "", sessionOpts{stdout: io.Discard})
		if err != nil {
			t.Fatalf("Test: %v", err)
		}
		if report.OK() {
			t.Error("a source that does not compile is a failure")
		}
		if file := report.Files[0]; file.Err == nil || !strings.Contains(file.Err.Error(), "broken.ar") {
			t.Errorf("error = %v, want it to name the module", file.Err)
		}
	})
}

// The report is what the command prints, so its shape matters.
func TestWritesAReport(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "printd 1;\n")
	writeAt(t, dir, "src/main.test.ar", `ident a = 1;
assert(a equals 1, "holds");
assert(a equals 2, "does not hold");
`)

	out := &strings.Builder{}
	if _, err := tested(t, "", sessionOpts{stdout: out}); err != nil {
		t.Fatalf("Test: %v", err)
	}

	report := out.String()
	for _, want := range []string{"main.test.ar", "ok", "holds", "FAIL", "does not hold", "1 passed, 1 failed in 1 file"} {
		if !strings.Contains(report, want) {
			t.Errorf("report is missing %q:\n%s", want, report)
		}
	}
}

func TestReportMentionsFilesThatCouldNotRun(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "ident a = 1;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(a equals 1, "holds");`)
	writeAt(t, dir, "src/orphan.test.ar", `assert(1 equals 1, "orphan");`)

	out := &strings.Builder{}
	if _, err := tested(t, "", sessionOpts{stdout: out}); err != nil {
		t.Fatalf("Test: %v", err)
	}

	report := out.String()
	if !strings.Contains(report, "could not run") {
		t.Errorf("the summary should count what did not run:\n%s", report)
	}
	if !strings.Contains(report, "ERROR") {
		t.Errorf("the file should be marked:\n%s", report)
	}
}

// A test file with nothing in it is not a failure, but the report should say so rather
// than look like a pass with no work.
func TestReportsAFileWithNoAssertions(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "ident a = 1;\n")
	writeAt(t, dir, "src/main.test.ar", "ident b = 2;\n")

	out := &strings.Builder{}
	report, err := tested(t, "", sessionOpts{stdout: out})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if !report.OK() {
		t.Error("no assertions is not a failure")
	}
	if !strings.Contains(out.String(), "no assertions") {
		t.Errorf("the report should say the file had none:\n%s", out.String())
	}
}

func TestWriteReportWithoutAWriter(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "ident a = 1;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(a equals 1, "holds");`)

	if _, err := tested(t, "", sessionOpts{stdout: nil}); err != nil {
		t.Errorf("a nil writer should discard the report, not fail: %v", err)
	}
}
