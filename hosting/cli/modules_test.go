package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// A project on disk, since this is where the world is: the command reads files, and where it
// reads them from is the point of half these tests. project makes it the working directory,
// which is what module names resolve from.
func projectOf(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := project(t)
	for path, source := range files {
		writeAt(t, dir, filepath.FromSlash(path), source)
	}
	return dir
}

// run compiles and evaluates the entry of a project, and answers with what it printed. The
// entry is named the way somebody standing in the project would name it.
func run(t *testing.T, entry string) (string, error) {
	t.Helper()

	var stdout bytes.Buffer
	err := newSession(t, sessionOpts{stdout: &stdout}).Run(t.Context(), filepath.FromSlash(entry))
	return stdout.String(), err
}

// The example the design was argued on, run the way somebody would run it.
//
// Both files bind base, which is the collision that ended the last attempt at modules. The
// imported scope sums the one its own file declared, so the answer is 14 and not 7.
func TestAProgramOfSeveralModules(t *testing.T) {
	projectOf(t, map[string]string{
		"src/a/b.ar":  "ident base = 10;\nident add = defer { feed(0) + feed(1) + base; };",
		"src/main.ar": "use a/b as x;\nident base = 3;\nprintd x.add(base, 1);",
	})

	printed, err := run(t, "src/main.ar")
	if err != nil {
		t.Fatalf("running: %v", err)
	}
	if strings.TrimSpace(printed) != "14" {
		t.Errorf("printed %q, want 14", printed)
	}
}

// A module runs once however many files name it, and it runs before whoever needs it.
func TestAModuleRunsOnceAndFirst(t *testing.T) {
	projectOf(t, map[string]string{
		"src/shared.ar": "printd 1;\nident v = 9;",
		"src/left.ar":   "use shared as s;\nident a = s.v;",
		"src/right.ar":  "use shared as s;\nident b = s.v;",
		"src/main.ar":   "use left as l;\nuse right as r;\nprintd 2;",
	})

	printed, err := run(t, "src/main.ar")
	if err != nil {
		t.Fatalf("running: %v", err)
	}
	if got := strings.Fields(printed); len(got) != 2 || got[0] != "1" || got[1] != "2" {
		t.Errorf("printed %q, want the module once and before the entry", printed)
	}
}

// What a program of one file always did, it still does: nothing looks for a module it never
// named.
func TestAProgramOfOneFileNeedsNoProject(t *testing.T) {
	projectOf(t, map[string]string{"loose.ar": "printd 1 + 1;"})

	printed, err := run(t, "loose.ar")
	if err != nil {
		t.Fatalf("running: %v", err)
	}
	if strings.TrimSpace(printed) != "2" {
		t.Errorf("printed %q, want 2", printed)
	}
}

// The refusals a person actually hits, and what they are told.
func TestWhatARefusalSays(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{
			name:  "a module that is not there",
			files: map[string]string{"src/main.ar": "use a/b as x;\nprintd 1;"},
			want:  []string{"module a/b is not there", "a/b.ar"},
		},
		{
			name: "a name the module does not have",
			files: map[string]string{
				"src/a/b.ar":  "ident base = 10;",
				"src/main.ar": "use a/b as x;\nprintd x.area;",
			},
			want: []string{"module a/b has no area", "it has base"},
		},
		{
			name: "two modules in a circle",
			files: map[string]string{
				"src/one.ar":  "use two as t;\nident a = 1;",
				"src/two.ar":  "use one as o;\nident b = 2;",
				"src/main.ar": "use one as o;\nprintd 1;",
			},
			want: []string{"circle", "one → two → one"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projectOf(t, tc.files)
			_, err := run(t, "src/main.ar")
			if err == nil {
				t.Fatal("expected a refusal")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %q, want it to say %q", err, want)
				}
			}
		})
	}
}

// A test runs against the source it belongs to and everything the two of them import.
//
// The two files are one module written twice, so the test calls what the source declared by
// the name the source typed, and the module the source brought in answers under its alias.
func TestATestSeesWhatItsSourceImports(t *testing.T) {
	projectOf(t, map[string]string{
		"src/geometry.ar":  "ident area = defer { feed(0) * feed(1); };",
		"src/main.ar":      "use geometry as g;\nident box = defer { g.area(feed(0), feed(1)); };",
		"src/main.test.ar": "assert(box(3, 4) equals 12, \"a box is its sides multiplied\");",
	})

	report, err := tested(t, "", sessionOpts{asserts: true})
	if err != nil {
		t.Fatalf("testing: %v", err)
	}
	if !report.OK() {
		t.Errorf("the run did not pass: %+v", report.Files)
	}
	if report.Passed != 1 {
		t.Errorf("%d assertions held, want 1", report.Passed)
	}
}

// And a test names the modules it uses like any other file, including one its source never
// mentioned.
func TestATestImportsOnItsOwn(t *testing.T) {
	projectOf(t, map[string]string{
		"src/numbers.ar":   "ident double = defer { feed(0) * 2; };",
		"src/main.ar":      "ident v = 21;",
		"src/main.test.ar": "use numbers as n;\nassert(n.double(v) equals 42, \"doubling what the source bound\");",
	})

	report, err := tested(t, "", sessionOpts{asserts: true})
	if err != nil {
		t.Fatalf("testing: %v", err)
	}
	if !report.OK() {
		t.Errorf("the run did not pass: %+v", report.Files)
	}
}

// A module that is not there is reported against the file that runs, rather than crashing the
// whole run.
func TestATestWhoseModuleIsNotThere(t *testing.T) {
	projectOf(t, map[string]string{
		"src/main.ar":      "use missing as m;\nident v = 1;",
		"src/main.test.ar": "assert(v equals 1, \"holds\");",
	})

	report, err := tested(t, "", sessionOpts{asserts: true})
	if err != nil {
		t.Fatalf("testing: %v", err)
	}
	if report.OK() {
		t.Fatal("the run passed with a module that is not there")
	}
	if len(report.Files) != 1 || report.Files[0].Err == nil {
		t.Fatalf("no file reported the refusal: %+v", report.Files)
	}
	if got := report.Files[0].Err.Error(); !strings.Contains(got, "module missing is not there") {
		t.Errorf("the error says %q", got)
	}
}
