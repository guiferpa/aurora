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
			// A shape of another module is not a value there, any more than a local one is
			// here: it is built, or it names what a value is read as. This used to be
			// refused for the module having no such name, because the name did not cross;
			// it crosses now, and the refusal says what it is instead. What both versions
			// prevent is the same — a name being loaded at run time that nobody bound.
			name: "a shape of another module is not a value",
			files: map[string]string{
				"src/a/b.ar":  "shape Prime { p };\nident v = 1;",
				"src/main.ar": "use a/b as x;\nprintd x.Prime;",
			},
			want: []string{"Prime is a shape of module a/b", "build a value with Prime{...}"},
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

// A test names what it checks, and what it checks names what it needs: a chain of modules
// loads before the test, each one once.
func TestATestReachesThroughAChainOfModules(t *testing.T) {
	projectOf(t, map[string]string{
		"src/geometry.ar":   "ident area = defer { feed(0) * feed(1); };",
		"src/boxes.ar":      "use geometry as g;\nident box = defer { g.area(feed(0), feed(1)); };",
		"src/main.ar":       "printd 1;",
		"src/boxes.test.ar": "use boxes as b;\nassert(b.box(3, 4) equals 12, \"a box is its sides multiplied\");",
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

// And it names as many as it needs, which is what it has instead of a file it belongs to.
func TestATestImportsEverythingItNeeds(t *testing.T) {
	projectOf(t, map[string]string{
		"src/numbers.ar":   "ident double = defer { feed(0) * 2; };",
		"src/values.ar":    "ident v = 21;",
		"src/main.ar":      "printd 1;",
		"src/main.test.ar": "use numbers as n;\nuse values as v;\nassert(n.double(v.v) equals 42, \"doubling what another module bound\");",
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
		"src/main.ar":      "printd 1;",
		"src/main.test.ar": "use missing as m;\nassert(1 equals 1, \"holds\");",
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

// What crosses a module and what does not, which is a sharper line than it looks.
//
// The value crosses whole: a shape is a run of tapes and nothing in it says a shape built
// it, so a scope in another file answers with one and the bytes arrive intact. What does not
// cross is the declaration — the table that turns a field name into an index — because it is
// read while a file is parsed and belongs to that file.
func TestAShapeValueCrossesAModuleButItsDeclarationDoesNot(t *testing.T) {
	files := map[string]string{
		"src/a/b.ar":  "shape Result { failed, value };\nident make = defer { Result{1, 42}; };",
		"src/main.ar": "use a/b as x;\nprintd x.make();",
	}

	t.Run("the value arrives whole", func(t *testing.T) {
		projectOf(t, files)
		printed, err := run(t, "src/main.ar")
		if err != nil {
			t.Fatalf("running: %v", err)
		}
		if strings.TrimSpace(printed) != "1 42" {
			t.Errorf("printed %q, want the two tapes the module built", printed)
		}
	})

	t.Run("and the field name has nowhere to be resolved", func(t *testing.T) {
		reading := map[string]string{
			"src/a/b.ar":  files["src/a/b.ar"],
			"src/main.ar": "use a/b as x;\nident r = x.make();\nprintd r.value;",
		}
		projectOf(t, reading)
		if _, err := run(t, "src/main.ar"); err == nil {
			t.Fatal("the field was read without a shape")
		} else if !strings.Contains(err.Error(), "nothing says which shape this value is") {
			t.Errorf("error = %q", err)
		}
	})
}

// A shape crosses a module with the promise that names it.
//
// The shape is declared in one file and read in another, and nothing in the reading file
// names it: what crossed is the promise the scope made, with the fields of the shape it
// answers with — which is what turns a field name into the index of a tape.
func TestAPromisedShapeCrossesAModule(t *testing.T) {
	projectOf(t, map[string]string{
		"src/os.ar": "shape Env { found, value };\n" +
			"ident lookup = defer {\n" +
			"  if feed(0) equals 0 { Env{0, 0}; } else { Env{1, 42}; };\n" +
			"} returns Env;",
		"src/main.ar": "use os as o;\n" +
			"ident r = o.lookup(1);\n" +
			"printd r.found;\nprintd r.value;\n" +
			"printd o.lookup(0).found;",
	})

	printed, err := run(t, "src/main.ar")
	if err != nil {
		t.Fatalf("running: %v", err)
	}
	if got := strings.Fields(printed); len(got) != 3 || got[0] != "1" || got[1] != "42" || got[2] != "0" {
		t.Errorf("printed %q, want 1 42 0", printed)
	}
}

// A field the promised shape does not have is refused where it was written, naming what the
// shape is made of — the same refusal a local shape gets.
func TestAFieldThePromiseDoesNotHave(t *testing.T) {
	projectOf(t, map[string]string{
		"src/os.ar":   "shape Env { found, value };\nident lookup = defer { Env{1, 42}; } returns Env;",
		"src/main.ar": "use os as o;\nident r = o.lookup(1);\nprintd r.missing;",
	})

	_, err := run(t, "src/main.ar")
	if err == nil {
		t.Fatal("a field nobody declared was read")
	}
	if !strings.Contains(err.Error(), "has no field named missing") {
		t.Errorf("error = %q", err)
	}
}

// A scope that promised nothing hands over a run of tapes and no shape, which is what it did
// before any of this: the reading file has to name one, and there is nothing to name.
func TestAScopeThatPromisedNothingCrossesNothing(t *testing.T) {
	projectOf(t, map[string]string{
		"src/os.ar":   "shape Env { found, value };\nident lookup = defer { Env{1, 42}; };",
		"src/main.ar": "use os as o;\nident r = o.lookup(1);\nprintd r.found;",
	})

	_, err := run(t, "src/main.ar")
	if err == nil {
		t.Fatal("a field was read off a scope that promised nothing")
	}
	if !strings.Contains(err.Error(), "nothing says which shape this value is") {
		t.Errorf("error = %q", err)
	}
}

// A test reads a promised shape like anybody else, because it imports like anybody else.
func TestATestReadsAPromisedShape(t *testing.T) {
	projectOf(t, map[string]string{
		"src/os.ar":        "shape Env { found, value };\nident lookup = defer { Env{1, 42}; } returns Env;",
		"src/main.ar":      "printd 1;",
		"src/main.test.ar": "use os as o;\nident r = o.lookup(0);\nassert(r.value equals 42, \"the field is read through the promise\");",
	})

	report, err := tested(t, "", sessionOpts{asserts: true})
	if err != nil {
		t.Fatalf("testing: %v", err)
	}
	if !report.OK() {
		t.Errorf("the run did not pass: %+v", report.Files)
	}
}

// Naming a shape another module declared: built, claimed with `as`, and promised with
// `returns` — the three places a shape's name is written, all reading the qualified form the
// way a qualified value already did.
func TestNamingAShapeOfAnotherModule(t *testing.T) {
	const geometry = "shape Square { width, height };\nident area = defer { ident s = feed(0) as Square; s.width * s.height; };"

	for _, tc := range []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "built here and fed there",
			source: "use geometry as g;\nprintd g.area(g.Square{4, 5});",
			want:   "20",
		},
		{
			name:   "claimed with as",
			source: "use geometry as g;\nident s = g.Square{6, 7} as g.Square;\nprintd s.height;",
			want:   "7",
		},
		{
			name: "promised with returns",
			source: "use geometry as g;\n" +
				"ident make = defer { g.Square{2, 3}; } returns g.Square;\n" +
				"ident s = make();\nprintd s.width;",
			want: "2",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projectOf(t, map[string]string{"src/geometry.ar": geometry, "src/main.ar": tc.source})
			printed, err := run(t, "src/main.ar")
			if err != nil {
				t.Fatalf("running: %v", err)
			}
			if strings.TrimSpace(printed) != tc.want {
				t.Errorf("printed %q, want %q", printed, tc.want)
			}
		})
	}
}

// And a shape a module does not have is refused where it was named, saying which module was
// asked — rather than with a qualified name nobody typed.
func TestAShapeAModuleDoesNotHave(t *testing.T) {
	projectOf(t, map[string]string{
		"src/geometry.ar": "shape Square { width, height };",
		"src/main.ar":     "use geometry as g;\nident s = 1 as g.Circle;",
	})

	_, err := run(t, "src/main.ar")
	if err == nil {
		t.Fatal("a shape nobody declared was named")
	}
	if !strings.Contains(err.Error(), "module geometry has no shape named Circle") {
		t.Errorf("error = %q", err)
	}
}

// A shape declared here and one of the same name there are two shapes, because the name that
// crosses is written the way an identifier of that module is — which nobody can type.
func TestTwoShapesOfTheSameNameAreTwoShapes(t *testing.T) {
	projectOf(t, map[string]string{
		"src/geometry.ar": "shape Square { width, height };\nident make = defer { Square{3, 4}; } returns Square;",
		"src/main.ar": "use geometry as g;\nshape Square { side };\n" +
			"ident mine = Square{9};\nident theirs = g.make();\n" +
			"printd mine.side;\nprintd theirs.height;",
	})

	printed, err := run(t, "src/main.ar")
	if err != nil {
		t.Fatalf("running: %v", err)
	}
	if got := strings.Fields(printed); len(got) != 2 || got[0] != "9" || got[1] != "4" {
		t.Errorf("printed %q, want 9 and 4", printed)
	}
}
