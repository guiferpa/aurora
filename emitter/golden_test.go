package emitter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

// A program touching most of the language, and the instructions it compiles to, written
// down. Nothing else pins the emitter's output as a whole: the other tests check one node
// at a time, so a change in how they are laid out — an extra save, a label that moves, a
// jump counted differently — passes them and shows up here.
//
// The file was produced before EmitInstruction was split into one function per node, so it
// is what proves that refactor changed nothing. Any diff after that is a real change to the
// compiled program, and updating this file is how you say you meant it.
func TestEmittedProgramMatchesTheGolden(t *testing.T) {
	dir := "testdata"

	source, err := os.ReadFile(filepath.Join(dir, "wide.ar"))
	if err != nil {
		t.Fatalf("reading the source: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(dir, "wide.ir"))
	if err != nil {
		t.Fatalf("reading the golden: %v", err)
	}

	tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens(source)
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	tree, err := parser.New(parser.NewParserOptions{}).Parse(parser.ParseInput{Filename: "wide.ar", Tokens: tokens})
	if err != nil {
		t.Fatalf("parser: %v", err)
	}
	insts, err := New(NewEmitterOptions{}).Emit(tree)
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}

	if got := ir.Format(insts); got != string(want) {
		t.Errorf("the emitted program no longer matches testdata/wide.ir.\n\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// The golden has to be able to fail, or it pins nothing: an emitter that answered with an
// empty program would pass a test that only checked it did not error.
func TestGoldenIsNotEmpty(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "wide.ir"))
	if err != nil {
		t.Fatal(err)
	}
	if len(want) == 0 {
		t.Fatal("the golden is empty, so it pins nothing")
	}
}
