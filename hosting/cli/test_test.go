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

func TestSourceForTest(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "greeting.test.ar", want: "greeting.ar"},
		{in: "src/utils/text.test.ar", want: filepath.FromSlash("src/utils/text.ar")},
		{in: "/abs/x.test.ar", want: filepath.FromSlash("/abs/x.ar")},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := SourceForTest(tc.in); got != tc.want {
				t.Errorf("SourceForTest(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// A test sees what the source of the same name declared, because both run in one
// evaluator, with the source first.
func TestRunsAgainstTheSourceItBelongsTo(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "ident double = defer { feed(0) * 2; };\nident base = 10;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(double(4) equals 8, "doubles");
assert(base equals 10, "sees the binding too");
`)

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if report.Passed != 2 || report.Failed != 0 {
		t.Errorf("got %d passed and %d failed, want 2 and 0", report.Passed, report.Failed)
	}
	if !report.OK() {
		t.Error("the report should be a pass")
	}
}

// The search starts at the directory of the profile's source and goes down. Anything above
// it belongs to another part of the project.
func TestSearchesFromTheSourceDirectoryDown(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "ident a = 1;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(a equals 1, "root of the search");`)
	writeAt(t, dir, "src/utils/text.ar", "ident b = 2;\n")
	writeAt(t, dir, "src/utils/text.test.ar", `assert(b equals 2, "a leaf");`)
	// Above the starting point, so out of reach.
	writeAt(t, dir, "outside.ar", "ident c = 3;\n")
	writeAt(t, dir, "outside.test.ar", `assert(c equals 4, "must not run");`)

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
	writeAt(t, dir, "src/main.ar", "ident a = 1;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(a equals 1, "one");`)
	writeAt(t, dir, "src/other.ar", "ident b = 2;\n")
	writeAt(t, dir, "src/other.test.ar", `assert(b equals 2, "two");`)

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
	writeAt(t, dir, "src/main.ar", "ident a = 1;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(a equals 1, "holds");
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

// A test cannot run without the source it belongs to: it would not see the code it checks.
func TestFailsWithoutItsSource(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "ident a = 1;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(a equals 1, "fine");`)
	writeAt(t, dir, "src/orphan.test.ar", `assert(1 equals 1, "no source");`)

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if report.OK() {
		t.Error("a test without its source is a failure")
	}

	var found bool
	for _, file := range report.Files {
		if strings.Contains(file.Path, "orphan") {
			found = true
			if file.Err == nil {
				t.Fatal("expected an error for the orphan")
			}
			if !strings.Contains(file.Err.Error(), "orphan.ar") {
				t.Errorf("error = %q, want it to name the missing file", file.Err)
			}
		}
	}
	if !found {
		t.Error("the orphan should appear in the report")
	}
}

// The two files share one scope, so a name declared in both is a conflict rather than a
// shadow. It is the documented consequence of the test seeing the source.
func TestNameDeclaredInBothFilesConflicts(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "ident a = 1;\n")
	writeAt(t, dir, "src/main.test.ar", "ident a = 2;\nassert(a equals 2, \"never gets here\");\n")

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if report.OK() {
		t.Fatal("the conflict should have been reported")
	}
	if file := report.Files[0]; file.Err == nil || !strings.Contains(file.Err.Error(), "conflict between identifiers") {
		t.Errorf("error = %v, want the identifier conflict", file.Err)
	}
}

// A test sees what its source declared, and a shape is declared while parsing rather than
// while running. The two files used to be parsed with a set of declarations each, so a test
// could call a defer from its source but could not build a shape it declared: `Point{1, 2}`
// stopped at the brace, because the name meant nothing to that parse.
func TestSeesAShapeDeclaredInItsSource(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "shape Point { x, y };\n")
	writeAt(t, dir, "src/main.test.ar",
		"ident p = Point{10, 20};\nassert(p.y equals 20, \"a shape crosses the pair\");\n")

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if file := report.Files[0]; file.Err != nil {
		t.Fatalf("the pair did not compile: %v", file.Err)
	}
	if !report.OK() {
		t.Errorf("the assertion did not hold: %+v", report.Files[0].Results)
	}
}

// The other way round is not true: what the test declares is its own. The source is compiled
// before the test exists as far as the parser is concerned, and a source that leaned on its
// test would not compile under "aurora run".
func TestASourceDoesNotSeeWhatItsTestDeclares(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "ident p = Point{1, 2};\n")
	writeAt(t, dir, "src/main.test.ar", "shape Point { x, y };\nassert(1 equals 1, \"unreached\");\n")

	report, err := tested(t, "", sessionOpts{stdout: io.Discard})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if file := report.Files[0]; file.Err == nil {
		t.Fatal("the source built a shape that only its test declares")
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
	writeAt(t, dir, "src/main.ar", "ident a = 200;\n")
	// 200 + 100 wraps to 44 on a one-byte tape.
	writeAt(t, dir, "src/main.test.ar", `assert(a + 100 equals 44, "wraps at the tape width");`)

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
	writeAt(t, dir, "src/main.ar", "ident a = 200;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(a + 100 equals 44, "wraps only on one byte");`)

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
		writeAt(t, dir, "src/main.ar", "ident a = 1;\n")
		writeAt(t, dir, "src/main.test.ar", "assert(@@@);\n")

		report, err := tested(t, "", sessionOpts{stdout: io.Discard})
		if err != nil {
			t.Fatalf("Test: %v", err)
		}
		if report.OK() {
			t.Error("a file that does not compile is a failure")
		}
	})

	t.Run("a source that does not compile", func(t *testing.T) {
		dir := project(t)
		writeAt(t, dir, "src/main.ar", "ident = ;\n")
		writeAt(t, dir, "src/main.test.ar", `assert(1 equals 1, "never runs");`)

		report, err := tested(t, "", sessionOpts{stdout: io.Discard})
		if err != nil {
			t.Fatalf("Test: %v", err)
		}
		if report.OK() {
			t.Error("a source that does not compile is a failure")
		}
		if file := report.Files[0]; file.Err == nil || !strings.Contains(file.Err.Error(), "main.ar") {
			t.Errorf("error = %v, want it to name the source", file.Err)
		}
	})
}

// The report is what the command prints, so its shape matters.
func TestWritesAReport(t *testing.T) {
	dir := project(t)
	writeAt(t, dir, "src/main.ar", "ident a = 1;\n")
	writeAt(t, dir, "src/main.test.ar", `assert(a equals 1, "holds");
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
