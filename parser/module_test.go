package parser

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/token"
)

// parseIn compiles source as the given module, which is what decides how the names in it are
// written down.
func parseIn(t *testing.T, source, name string) (ast.AST, error) {
	t.Helper()
	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		return ast.AST{}, err
	}
	return New().Parse(ParseInput{Filename: "main.ar", Tokens: tokens, Module: name})
}

func mustParseIn(t *testing.T, source, name string) ast.AST {
	t.Helper()
	tree, err := parseIn(t, source, name)
	if err != nil {
		t.Fatalf("parsing %q as %s: %v", source, name, err)
	}
	return tree
}

// Inside a module every name carries the module in front of it — binding and mention alike,
// which is what lets them go on finding each other.
func TestNamesInAModuleCarryIt(t *testing.T) {
	tree := mustParseIn(t, "ident base = 10;\nprintd base;", "a/b")

	binding, ok := tree.Nodes[0].(ast.IdentLiteral)
	if !ok {
		t.Fatalf("the first node is %T", tree.Nodes[0])
	}
	if binding.Id != "a/b.base" {
		t.Errorf("bound %q, want %q", binding.Id, "a/b.base")
	}

	printed, ok := tree.Nodes[1].(ast.PrintStatement)
	if !ok {
		t.Fatalf("the second node is %T", tree.Nodes[1])
	}
	mention, ok := printed.Param.(ast.IdentifierLiteral)
	if !ok {
		t.Fatalf("what is printed is %T", printed.Param)
	}
	if mention.Value != "a/b.base" {
		t.Errorf("mentioned %q, want %q", mention.Value, "a/b.base")
	}
}

// The file somebody asked to run has no module, and its names are written as they were typed.
// It is what keeps every program that exists today compiling to the same instructions.
func TestNamesWithoutAModuleAreWhatWasTyped(t *testing.T) {
	tree := mustParseIn(t, "ident base = 10;", "")
	if binding := tree.Nodes[0].(ast.IdentLiteral); binding.Id != "base" {
		t.Errorf("bound %q, want %q", binding.Id, "base")
	}
}

// A qualified name is written the way the module writes it, and the alias is gone by then.
func TestAQualifiedNameIsWrittenAsTheModuleWritesIt(t *testing.T) {
	tree := mustParseIn(t, "use a/b as x;\nprintd x.area(2, 3);", "")

	printed := tree.Nodes[1].(ast.PrintStatement)
	call, ok := printed.Param.(ast.CalleeLiteral)
	if !ok {
		t.Fatalf("what is printed is %T, want a call", printed.Param)
	}
	if call.Id.Value != "a/b.area" {
		t.Errorf("called %q, want %q", call.Id.Value, "a/b.area")
	}
	if len(call.Params) != 2 {
		t.Errorf("called with %d values, want 2", len(call.Params))
	}
}

// Reaching a name without calling it is the same name.
func TestAQualifiedNameWithoutACall(t *testing.T) {
	tree := mustParseIn(t, "use a/b as x;\nprintd x.base;", "")
	printed := tree.Nodes[1].(ast.PrintStatement)
	mention, ok := printed.Param.(ast.IdentifierLiteral)
	if !ok {
		t.Fatalf("what is printed is %T, want a name", printed.Param)
	}
	if mention.Value != "a/b.base" {
		t.Errorf("read %q, want %q", mention.Value, "a/b.base")
	}
}

// Every qualified name leaves with the tree, because only whoever holds the other modules can
// say whether it is really there.
func TestQualifiedNamesLeaveWithTheTree(t *testing.T) {
	tree := mustParseIn(t, "use a/b as x;\nuse c as y;\nprintd x.area(1) + y.n;", "")

	if len(tree.References) != 2 {
		t.Fatalf("read %d qualified names, want 2: %+v", len(tree.References), tree.References)
	}
	for i, want := range []ast.Reference{{Module: "a/b", Symbol: "area"}, {Module: "c", Symbol: "n"}} {
		got := tree.References[i]
		if got.Module != want.Module || got.Symbol != want.Symbol {
			t.Errorf("reference %d = %s.%s, want %s.%s", i, got.Module, got.Symbol, want.Module, want.Symbol)
		}
		if got.Token == nil {
			t.Errorf("reference %d carries no token, so nothing can point at it", i)
		}
		if got.Name() != want.Module+"."+want.Symbol {
			t.Errorf("reference %d is called %q", i, got.Name())
		}
	}
}

// A struct is still a struct inside a module: its name never reaches an instruction and never
// leaves the file, so it is read as it was typed while everything around it is renamed.
func TestAStructInsideAModule(t *testing.T) {
	tree := mustParseIn(t, "struct Point { x, y };\nident p = Point{1, 2};\nprintd p.y;", "a/b")

	if binding := tree.Nodes[1].(ast.IdentLiteral); binding.Id != "a/b.p" {
		t.Errorf("bound %q, want %q", binding.Id, "a/b.p")
	}
	printed := tree.Nodes[2].(ast.PrintStatement)
	field, ok := printed.Param.(ast.FieldExpression)
	if !ok {
		t.Fatalf("what is printed is %T, want a field", printed.Param)
	}
	if field.Index != 1 {
		t.Errorf("read field %d, want 1", field.Index)
	}
}

// And it is named as it was typed when something goes wrong with it.
func TestAStructIsNamedAsItWasTyped(t *testing.T) {
	_, err := parseIn(t, "struct Point { x, y };\nprintd Point;", "a/b")
	if err == nil {
		t.Fatal("expected a compile error")
	}
	if !strings.Contains(err.Error(), "Point is a struct") || strings.Contains(err.Error(), "a/b.Point") {
		t.Errorf("error = %q, want it to name the struct as it was typed", err)
	}
}

// What an alias is not.
func TestAnAliasIsNotAValueAndNotAName(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   string
	}{
		{"printed", "use a/b as x;\nprintd x;", "x is the module a/b"},
		{"added", "use a/b as x;\nprintd x + 1;", "x is the module a/b"},
		{"bound over", "use a/b as x;\nident x = 1;", "x is already the alias of a/b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseIn(t, tc.source, "")
			if err == nil {
				t.Fatal("expected a compile error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to say %q", err, tc.want)
			}
			if positioned, ok := err.(*token.Error); !ok || positioned.Line == 0 {
				t.Errorf("error is %T and unpositioned", err)
			}
		})
	}
}
