package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/builder/evm"
	"github.com/guiferpa/aurora/wire/ir"
)

// Every program this compiler produces says what it means.
//
// The check is in wire/ir and knows nothing about how a program got there; this runs it over
// programs somebody would write, which is the only way to find out whether the emitter has
// been producing something a consumer had to be forgiving about. Nobody could ask before,
// because nothing in the IR was an assertion — the failure mode was a binary that deployed and
// was quietly wrong.
func TestEveryProgramTheCompilerProducesVerifies(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{name: "arithmetic", source: `ident f = defer { feed(0) + feed(1) * 2; };`},
		{name: "a name bound inside a scope", source: `ident f = defer { ident x = feed(0); x + x; };`},
		{name: "a branch", source: `ident f = defer { if feed(0) bigger 2 { 1; } else { 2; }; };`},
		{name: "a branch inside a branch", source: `ident f = defer { if feed(0) bigger 2 { if feed(0) bigger 4 { 1; } else { 2; }; } else { 3; }; };`},
		{name: "a branch statement", source: `ident f = defer { branch { feed(0) equals 0: 1, feed(0) equals 1: 2, 3; }; };`},
		{name: "a scope calling another", source: "ident g = defer { feed(0) * 2; };\nident f = defer { g(feed(0)) + 1; };"},
		{name: "a scope nested in a scope", source: `ident f = defer { ident g = defer { 1; }; 2; };`},
		{name: "a shape and a field", source: "shape P { a, b };\nident f = defer { ident p = P{feed(0), 2}; p.a + p.b; };"},
		{name: "a wide shape", source: "shape W { a, b, c, d, e };\nident f = defer { ident w = W{1, 2, 3, 4, feed(0)}; w.e; };"},
		{name: "the tape operations", source: `ident f = defer { ident x = feed(0); ident h = head x 1; ident t = tail x 1; h + t; };`},
		{name: "storage", source: `ident f = defer { sstore 1 feed(0); sload 1; };`},
		{name: "a value nothing takes", source: `ident f = defer { feed(0) + 1; feed(0) + 2; };`},
		{name: "the prints", source: `ident f = defer { printd feed(0); };`},
		{name: "an inline block", source: `ident f = defer { ident x = { feed(0) + 1; }; x + 1; };`},
		{name: "text in a tape", source: `ident f = defer { "ab"; };`},
		{name: "the top of a program", source: "printd 1 + 1;"},
		// The one the check found the day it was written. An empty block is worth the neutral
		// tape, and that used to be a reference to a label nothing carried — a value nobody
		// left. It answered zeros anyway, because a consumer that looked the name up found
		// nothing and nothing is zeros, which is working by accident.
		{name: "an empty block", source: "ident empty = { };\nprintd empty;"},
		{name: "an empty block inside a scope", source: "ident f = defer { ident e = { }; e + feed(0); };"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeAt(t, dir, "contract.ar", tc.source)
			program, err := newSession(t, sessionOpts{}).compile(path)
			if err != nil {
				t.Fatalf("compiling: %v", err)
			}

			for _, problem := range ir.Verify(program.Blocks) {
				t.Errorf("%v", problem)
			}
		})
	}
}

// A program that does not say what it means gets no bytecode.
//
// It is checked as it arrives rather than after the lowering, and the difference is not
// pedantry: what comes out of the lowering is a schedule for one machine, where a binding a
// scope ends with leaves nothing on the stack and a push is written under that same name to
// stand in for it. Two values under one name is exactly what the check refuses, and it is
// right to — of the IR. Of a schedule it is not a question.
func TestABuildRefusesAnIRThatDoesNotHold(t *testing.T) {
	blocks := []ir.Block{{
		ID:    0,
		Insts: []ir.Instruction{ir.NewInstruction([]byte("01"), ir.OpAdd, ir.RefTo([]byte("ff")), ir.Nothing())},
		Term:  ir.Ends(ir.RefTo([]byte("01"))),
	}}

	_, err := evm.NewBuilder(blocks, evm.NewBuilderOptions{TapeSize: 8}).Build()
	if err == nil {
		t.Fatal("it wrote bytecode for a program that reads a value nobody leaves")
	}
	if !strings.Contains(err.Error(), "does not say what it means") {
		t.Errorf("it says %q", err)
	}
	if !strings.Contains(err.Error(), "block 0") {
		t.Errorf("it says %q, and never says where", err)
	}
}

// And a scope that ends with a binding still builds, which is the case that made the check
// run where it does.
func TestAScopeEndingWithABindingStillBuilds(t *testing.T) {
	dir := t.TempDir()
	path := writeAt(t, dir, "contract.ar", "ident f = defer { ident x = feed(0); ident y = x + 1; };\n")

	if _, err := newSession(t, sessionOpts{}).Build(t.Context(), path, filepath.Join(dir, "out.bin")); err != nil {
		t.Fatalf("building: %v", err)
	}
}
