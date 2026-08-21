package emitter

import (
	"testing"

	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/ir"
	"github.com/guiferpa/aurora/wire/token"
)

// Where a symbol comes from: the node, and not the token it was read at.
//
// For everything the parser writes today the two are the same text, so no program compiled
// from source can tell them apart — the golden does that job, and it not moving is what says
// this changed no output. What needs a test of its own is the case that does not exist yet: a
// name the compiler wrote rather than read, which is what a module prefix will be. Below the
// token says "area" and the node says "geometry.area", and the instruction has to carry the
// second.
func symbolOf(t *testing.T, insts []ir.Instruction, opcode byte) string {
	t.Helper()
	for _, inst := range insts {
		if inst.GetOpCode() == opcode {
			return string(inst.GetLeft().Bytes())
		}
	}
	t.Fatalf("no %s was emitted", ir.ResolveOpCode(opcode))
	return ""
}

func TestSymbolComesFromTheNodeAndNotItsToken(t *testing.T) {
	read := token.New([]byte("area"), token.TagId, 1, 1, 0)
	const written = "geometry.area"

	insts, err := New(NewEmitterOptions{}).Emit(ast.AST{Filename: "main.ar", Nodes: []ast.Node{
		ast.IdentLiteral{Id: written, Token: read, Value: ast.NumberLiteral{Value: 1}},
		ast.IdentifierLiteral{Value: written, Token: read},
		ast.CalleeLiteral{Id: ast.IdentifierLiteral{Value: written, Token: read}},
	}})
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}

	for _, opcode := range []byte{ir.OpIdent, ir.OpLoad, ir.OpCall} {
		if got := symbolOf(t, insts, opcode); got != written {
			t.Errorf("%s carries %q, want %q", ir.ResolveOpCode(opcode), got, written)
		}
	}
}

// And a node with no token at all still compiles, which is the other half of the same
// statement: the token is carried for position, and nothing reads it to find out what a name
// is. A synthesised node has nowhere to have been read from.
func TestANodeWithoutATokenStillCarriesItsName(t *testing.T) {
	insts, err := New(NewEmitterOptions{}).Emit(ast.AST{Filename: "main.ar", Nodes: []ast.Node{
		ast.IdentLiteral{Id: "a", Value: ast.NumberLiteral{Value: 1}},
		ast.IdentifierLiteral{Value: "a"},
	}})
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}

	if got := symbolOf(t, insts, ir.OpIdent); got != "a" {
		t.Errorf("OpIdent carries %q, want %q", got, "a")
	}
	if got := symbolOf(t, insts, ir.OpLoad); got != "a" {
		t.Errorf("OpLoad carries %q, want %q", got, "a")
	}
}

// A declaration does no work, and an import is one: it names a module the compiler resolves
// before the emitter ever runs, so what comes out is the neutral value and nothing else. The
// same form as a shape declaration, and for the same reason.
func TestAnImportEmitsNoWork(t *testing.T) {
	program := compile(t, "use a/b/c as x;")

	if len(program.Instructions) != 1 {
		t.Fatalf("an import emitted %d instructions, want 1", len(program.Instructions))
	}
	if got := program.Instructions[0].GetOpCode(); got != ir.OpSave {
		t.Errorf("an import emitted %s, want OpSave of the neutral value", ir.ResolveOpCode(got))
	}
}
