package resolver

import (
	"fmt"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
	"github.com/guiferpa/aurora/wire/token"
)

// The resolver never opens a file: it asks the port it was handed. So does every test here,
// and that is the point of them — a test that needed a directory on disk would be saying the
// port is in the wrong place.

// files is a project: the path a module would be read from, and what is in it.
type files map[string]string

// resolve runs the resolver over files, and answers with what it found and what it asked to
// read, in the order it asked.
func resolve(t *testing.T, root, entry string, sources files) ([]module.Module, []string, error) {
	t.Helper()

	asked := make([]string, 0)
	resolver := New(Options{
		SourceRoot: root,
		Read: func(path string) ([]byte, error) {
			asked = append(asked, path)
			source, ok := sources[path]
			if !ok {
				return nil, fmt.Errorf("no such file")
			}
			return []byte(source), nil
		},
		Parse: func(filename string, _ module.ID, source []byte, imports map[string]ast.Offer) (ast.AST, error) {
			tokens, err := lexer.New().GetFilledTokens(source)
			if err != nil {
				return ast.AST{}, err
			}
			return parser.New().Parse(parser.ParseInput{Filename: filename, Tokens: tokens, Imports: imports})
		},
		Header: header,
	})

	modules, err := resolver.Resolve(entry)
	return modules, asked, err
}

// ids is the order that came back, as names, with the entry showing as the empty one.
func ids(modules []module.Module) []string {
	names := make([]string, 0, len(modules))
	for _, each := range modules {
		names = append(names, string(each.ID))
	}
	return names
}

func mustResolve(t *testing.T, root, entry string, sources files) []module.Module {
	t.Helper()
	modules, _, err := resolve(t, root, entry, sources)
	if err != nil {
		t.Fatalf("resolving: %v", err)
	}
	return modules
}

// A file that needs nothing is a program of one module, and it is the entry: no name, because
// nothing imports it.
func TestAProgramOfOneFile(t *testing.T) {
	modules := mustResolve(t, "src", "src/main.ar", files{
		"src/main.ar": "printd 1;",
	})

	if got := ids(modules); len(got) != 1 || got[0] != "" {
		t.Fatalf("order = %q, want the entry alone", got)
	}
	if !modules[0].IsEntry() {
		t.Error("the file that was asked for is not the entry")
	}
}

// What a file needs comes before it, and the entry is last: its body runs after everything it
// depends on has run.
func TestDependenciesComeFirst(t *testing.T) {
	modules := mustResolve(t, "src", "src/main.ar", files{
		"src/main.ar":     "use a/b as x;\nprintd 1;",
		"src/a/b.ar":      "use deep/one as d;\nident v = 1;",
		"src/deep/one.ar": "ident w = 2;",
	})

	want := []string{"deep/one", "a/b", ""}
	if got := ids(modules); !equal(got, want) {
		t.Errorf("order = %q, want %q", got, want)
	}
}

// A module two files name is read once. Its body is a program that runs, so reading it twice
// would run it twice.
func TestAModuleIsReadOnce(t *testing.T) {
	modules, asked, err := resolve(t, "src", "src/main.ar", files{
		"src/main.ar":   "use left as l;\nuse right as r;\nprintd 1;",
		"src/left.ar":   "use shared as s;\nident a = 1;",
		"src/right.ar":  "use shared as s;\nident b = 2;",
		"src/shared.ar": "ident c = 3;",
	})
	if err != nil {
		t.Fatalf("resolving: %v", err)
	}

	want := []string{"shared", "left", "right", ""}
	if got := ids(modules); !equal(got, want) {
		t.Errorf("order = %q, want %q", got, want)
	}

	read := 0
	for _, path := range asked {
		if path == "src/shared.ar" {
			read++
		}
	}
	if read != 1 {
		t.Errorf("src/shared.ar was read %d times, want 1", read)
	}
}

// The same module under two names in the same file is still one module.
func TestTwoAliasesForOneModule(t *testing.T) {
	modules, asked, err := resolve(t, "src", "src/main.ar", files{
		"src/main.ar":  "use thing as a;\nuse thing as b;\nprintd 1;",
		"src/thing.ar": "ident v = 1;",
	})
	if err != nil {
		t.Fatalf("resolving: %v", err)
	}
	if got := ids(modules); !equal(got, []string{"thing", ""}) {
		t.Errorf("order = %q, want thing and the entry", got)
	}
	if len(asked) != 2 {
		t.Errorf("read %d files, want the entry and thing once each: %q", len(asked), asked)
	}
}

// The source root is where a module name resolves from, and the name says the rest.
func TestTheSourceRootDecidesWhereToLook(t *testing.T) {
	modules, asked, err := resolve(t, "lib", "lib/main.ar", files{
		"lib/main.ar":           "use deep/down/here as d;\nprintd 1;",
		"lib/deep/down/here.ar": "ident v = 1;",
	})
	if err != nil {
		t.Fatalf("resolving: %v", err)
	}
	if got := ids(modules); !equal(got, []string{"deep/down/here", ""}) {
		t.Errorf("order = %q", got)
	}
	if want := "lib/deep/down/here.ar"; asked[1] != want {
		t.Errorf("read %q, want %q", asked[1], want)
	}
}

// Everything that is not a program, and what it says about it.
var refusals = []struct {
	name    string
	entry   string
	sources files
	want    []string
}{
	{
		name:    "a module that is not there",
		entry:   "src/main.ar",
		sources: files{"src/main.ar": "use a/b as x;\nprintd 1;"},
		want:    []string{"a/b is not there", "src/a/b.ar"},
	},
	{
		name:  "two modules in a circle",
		entry: "src/main.ar",
		sources: files{
			"src/main.ar": "use one as o;\nprintd 1;",
			"src/one.ar":  "use two as t;\nident a = 1;",
			"src/two.ar":  "use one as o;\nident b = 2;",
		},
		want: []string{"circle", "one → two → one"},
	},
	{
		name:  "three modules in a circle",
		entry: "src/main.ar",
		sources: files{
			"src/main.ar":  "use one as o;\nprintd 1;",
			"src/one.ar":   "use two as t;\nident a = 1;",
			"src/two.ar":   "use three as t;\nident b = 2;",
			"src/three.ar": "use one as o;\nident c = 3;",
		},
		want: []string{"one → two → three → one"},
	},
	{
		name:  "a module that imports itself",
		entry: "src/main.ar",
		sources: files{
			"src/main.ar": "use one as o;\nprintd 1;",
			"src/one.ar":  "use one as o;\nident a = 1;",
		},
		want: []string{"one → one"},
	},
	{
		// It would be read twice, under two names, and its top level would run twice.
		name:    "the program importing itself",
		entry:   "src/main.ar",
		sources: files{"src/main.ar": "use main as m;\nprintd 1;"},
		want:    []string{"the file being run", "cannot import itself"},
	},
}

func TestRefusals(t *testing.T) {
	for _, tc := range refusals {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := resolve(t, "src", tc.entry, tc.sources)
			if err == nil {
				t.Fatal("expected a refusal")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %q, want it to say %q", err, want)
				}
			}
			positioned, ok := err.(*token.Error)
			if !ok {
				t.Fatalf("error is %T, want a positioned *token.Error so an editor can underline it", err)
			}
			if positioned.Line == 0 || positioned.Column == 0 {
				t.Errorf("error has no position: line %d, column %d", positioned.Line, positioned.Column)
			}
		})
	}
}

// The entry not being readable is not about a module, so it is not underlined at a use.
func TestTheEntryNotBeingThere(t *testing.T) {
	_, _, err := resolve(t, "src", "src/main.ar", files{})
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if !strings.Contains(err.Error(), "src/main.ar") {
		t.Errorf("error = %q, want it to name the file", err)
	}
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// header reads what a source imports without parsing it, which is what lets a module be read
// before whoever imports it.
func header(source []byte) ([]ast.UseDeclaration, error) {
	tokens, err := lexer.New().GetFilledTokens(source)
	if err != nil {
		return nil, err
	}
	return parser.ScanUses(tokens), nil
}
