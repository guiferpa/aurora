package loader

import (
	"fmt"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/resolver"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/ir"
	"github.com/guiferpa/aurora/wire/module"
	"github.com/guiferpa/aurora/wire/token"
)

// The whole path, in memory: the resolver finds the files, the parser writes each module's
// names with the module in front of them, and the check here says whether a qualified name
// is really there. Nothing touches a disk.
func load(t *testing.T, sources map[string]string) ([]module.Module, error) {
	t.Helper()

	return resolver.New(resolver.Options{
		SourceRoot: "src",
		Read: func(path string) ([]byte, error) {
			source, ok := sources[path]
			if !ok {
				return nil, fmt.Errorf("no such file")
			}
			return []byte(source), nil
		},
		Parse: func(filename string, id module.ID, source []byte, imports map[string][]ast.Promise) (ast.AST, error) {
			tokens, err := lexer.New().GetFilledTokens(source)
			if err != nil {
				return ast.AST{}, err
			}
			return parser.New().Parse(parser.ParseInput{
				Filename: filename,
				Tokens:   tokens,
				Module:   string(id),
				Imports:  imports,
			})
		},
		Header: header,
	}).Resolve("src/main.ar")
}

// What a module offers is what it binds at the top, under the name its own file typed.
func TestWhatAModuleOffers(t *testing.T) {
	modules, err := load(t, map[string]string{
		"src/main.ar": "use a/b as x;\nprintd x.area(2, 3);",
		"src/a/b.ar": "struct Point { x, y };\n" +
			"ident base = 10;\n" +
			"ident area = defer { ident inner = 1; feed(0) * feed(1); };\n" +
			"printd 1;",
	})
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	got := Exports(modules[0])
	want := []string{"base", "area"}
	if len(got) != len(want) {
		t.Fatalf("exports = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("exports = %q, want %q", got, want)
		}
	}
}

// The name a module binds and the name another file asks for are the same text, which is the
// whole of the binding: two files arriving at one string.
func TestAQualifiedNameIsTheNameTheModuleBound(t *testing.T) {
	modules, err := load(t, map[string]string{
		"src/main.ar": "use a/b as x;\nprintd x.area(2, 3);",
		"src/a/b.ar":  "ident area = defer { feed(0) * feed(1); };",
	})
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if err := Check(modules); err != nil {
		t.Fatalf("checking: %v", err)
	}

	binding, ok := modules[0].Tree.Nodes[0].(ast.IdentLiteral)
	if !ok {
		t.Fatalf("the module binds %T", modules[0].Tree.Nodes[0])
	}
	if binding.Id != "a/b.area" {
		t.Errorf("the module bound %q, want %q", binding.Id, "a/b.area")
	}

	references := modules[1].Tree.References
	if len(references) != 1 {
		t.Fatalf("the entry read %d qualified names, want 1", len(references))
	}
	if got := module.Qualify(module.ID(references[0].Module), references[0].Symbol); got != binding.Id {
		t.Errorf("the entry asked for %q and the module bound %q", got, binding.Id)
	}
}

// The entry keeps the names it typed, so nothing that exists today is written differently.
func TestTheEntryIsNotPrefixed(t *testing.T) {
	modules, err := load(t, map[string]string{
		"src/main.ar": "use a/b as x;\nident base = 3;\nprintd base;",
		"src/a/b.ar":  "ident base = 10;",
	})
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	entry := modules[len(modules)-1]
	if !entry.IsEntry() {
		t.Fatal("the last module is not the entry")
	}
	if binding := entry.Tree.Nodes[1].(ast.IdentLiteral); binding.Id != "base" {
		t.Errorf("the entry bound %q, want %q", binding.Id, "base")
	}
	// And the two files binding the same word are two different names, which is what lets
	// them share one environ.
	if binding := modules[0].Tree.Nodes[0].(ast.IdentLiteral); binding.Id != "a/b.base" {
		t.Errorf("the module bound %q, want %q", binding.Id, "a/b.base")
	}
}

// A name that is not there is refused where it was asked for, saying what is.
func TestANameThatIsNotThere(t *testing.T) {
	modules, err := load(t, map[string]string{
		"src/main.ar": "use a/b as x;\nprintd x.perimeter(2, 3);",
		"src/a/b.ar":  "ident base = 10;\nident area = defer { feed(0); };",
	})
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	err = Check(modules)
	if err == nil {
		t.Fatal("expected a refusal")
	}
	for _, want := range []string{"module a/b has no perimeter", "it has area, base"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to say %q", err, want)
		}
	}
	positioned, ok := err.(*token.Error)
	if !ok {
		t.Fatalf("error is %T, want a positioned *token.Error so an editor can underline it", err)
	}
	if positioned.Line != 2 {
		t.Errorf("error is at line %d, want the line the name was asked on", positioned.Line)
	}
}

// A module that offers nothing says so, rather than listing an empty list.
func TestAModuleThatOffersNothing(t *testing.T) {
	modules, err := load(t, map[string]string{
		"src/main.ar": "use a/b as x;\nprintd x.area(1);",
		"src/a/b.ar":  "printd 1;",
	})
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if err := Check(modules); err == nil || !strings.Contains(err.Error(), "it has nothing") {
		t.Errorf("error = %v, want it to say the module has nothing", err)
	}
}

// A reference to a module nobody loaded cannot come out of a resolved program, so it is built
// by hand — the check is what stands between a mistake upstream and a name resolving to
// whatever happens to be around at run time.
func TestAReferenceToAModuleThatWasNeverLoaded(t *testing.T) {
	at := token.New([]byte("add"), token.TagId, 3, 7, 20)
	modules := []module.Module{{
		ID: "",
		Tree: ast.AST{
			Filename:   "main.ar",
			References: []ast.Reference{{Module: "a/b", Symbol: "add", Token: at}},
		},
	}}

	err := Check(modules)
	if err == nil || !strings.Contains(err.Error(), "never loaded") {
		t.Errorf("error = %v, want it to say the module was never loaded", err)
	}
}

// The ranges tile the stream: every instruction belongs to exactly one module, and they are
// laid down in the order the modules load, which is dependencies first.
func TestLoadLaysEveryModuleEndToEnd(t *testing.T) {
	modules, err := load(t, map[string]string{
		"src/main.ar": "use a/b as x;\nprintd x.area(2, 3);",
		"src/a/b.ar":  "ident area = defer { feed(0) * feed(1); };",
	})
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	program, err := Load(modules, emitter.New(emitter.NewEmitterOptions{}).EmitProgram)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(program.Ranges) != 2 {
		t.Fatalf("laid %d ranges, want one per module", len(program.Ranges))
	}
	if program.Ranges[0].Module != "a/b" || program.Ranges[1].Module != "" {
		t.Errorf("ranges are %q and %q, want the module before the entry",
			program.Ranges[0].Module, program.Ranges[1].Module)
	}
	if program.Ranges[0].Filename != "src/a/b.ar" {
		t.Errorf("the range names %q, want the file it came from", program.Ranges[0].Filename)
	}

	previous := uint64(0)
	for i, each := range program.Ranges {
		if each.From != previous {
			t.Errorf("range %d starts at %d, want %d", i, each.From, previous)
		}
		if each.To <= each.From {
			t.Errorf("range %d is empty", i)
		}
		previous = each.To
	}
	if previous != uint64(len(program.Instructions)) {
		t.Errorf("the ranges cover %d instructions, the program has %d", previous, len(program.Instructions))
	}
}

// A name that is not there stops the load, rather than being compiled into a program that
// would look for it while running.
func TestLoadRefusesBeforeItEmits(t *testing.T) {
	modules, err := load(t, map[string]string{
		"src/main.ar": "use a/b as x;\nprintd x.perimeter(1);",
		"src/a/b.ar":  "ident area = defer { feed(0); };",
	})
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	emitted := 0
	_, err = Load(modules, func(tree ast.AST) (ir.Program, error) {
		emitted++
		return ir.Program{}, nil
	})
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if emitted != 0 {
		t.Errorf("emitted %d modules before refusing, want none", emitted)
	}
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
