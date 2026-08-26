package emitter

import (
	"testing"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

// A block says how many tapes the value it returns is made of.
//
// A construction says its width by having that many operands and a field read carries the
// width of the run it comes out of, so the one place nothing said it was the value coming
// back from a call. That is a fact about the scope, which is why it is on the block.
func TestABlockSaysHowWideTheRunItReturnsIs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   int
	}{
		{
			name:   "a scope that declared",
			source: "shape P { a, b };\nident make = defer { P{1, 2}; } returns P;",
			want:   2,
		},
		{
			// The compiler works this out rather than waiting for a `returns`, so most
			// scopes that return a run say so without the file writing a word.
			name:   "a scope that declared nothing",
			source: "shape P { a, b, c };\nident make = defer { P{1, 2, 3}; };",
			want:   3,
		},
		{
			name:   "a scope that returns a tape",
			source: "ident add = defer { feed(0) + feed(1); };",
			want:   0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blocks := blocksOfSource(t, tc.source)

			widest := 0
			for _, block := range blocks {
				if block.Tapes > widest {
					widest = block.Tapes
				}
			}
			if widest != tc.want {
				t.Errorf("the widest block returns %d tapes, want %d", widest, tc.want)
			}
		})
	}
}

// The width lands on the scope's own block, and not on the one that binds it.
//
// Binding a scope to a name is an instruction of the block doing the binding; the run comes
// back from the block that computes it, and that is the one a caller has to copy from.
func TestTheWidthIsOnTheScopeAndNotOnWhoeverBoundIt(t *testing.T) {
	blocks := blocksOfSource(t, "shape P { a, b };\nident make = defer { P{1, 2}; };\nmake();")

	if len(blocks) < 2 {
		t.Fatalf("a program with a scope came out as %d blocks", len(blocks))
	}
	if blocks[0].Tapes != 0 {
		t.Errorf("the top of the program says %d tapes, and it returns no run", blocks[0].Tapes)
	}

	said := 0
	for _, block := range blocks[1:] {
		if block.Tapes == 2 {
			said++
		}
	}
	if said != 1 {
		t.Errorf("%d blocks say they return two tapes, want the one scope", said)
	}
}

func blocksOfSource(t *testing.T, source string) []ir.Block {
	t.Helper()
	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	tree, err := parser.New().Parse(parser.ParseInput{Filename: "main.ar", Tokens: tokens})
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	program, err := New(NewEmitterOptions{}).EmitProgram(tree)
	if err != nil {
		t.Fatalf("emitting: %v", err)
	}
	return program.Blocks
}
