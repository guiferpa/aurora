package trace

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

// compile runs the front end over source, which is what a host hands to trace.
func compile(t *testing.T, source string) (parser.AST, emitter.Program) {
	t.Helper()

	tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	ast, err := parser.New(parser.NewParserOptions{Filename: "main.ar", Tokens: tokens}).Parse()
	if err != nil {
		t.Fatalf("parser: %v", err)
	}
	program, err := emitter.New(emitter.NewEmitterOptions{}).EmitProgram(ast)
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}
	return ast, program
}

// The tree is shown with every node labelled by its type, which is the first thing anyone
// reading a tree looks for. Positions are left out: a token carries its whole location, and
// printing that drowns the tree it belongs to.
func TestASTNamesEveryNode(t *testing.T) {
	ast, _ := compile(t, "ident a = 1 + 2;\n")

	out := bytes.NewBuffer(nil)
	if err := AST(out, ast); err != nil {
		t.Fatalf("AST: %v", err)
	}

	written := out.String()
	for _, want := range []string{"IdentLiteral", "BinaryExpression", "NumberLiteral", `"type"`} {
		if !strings.Contains(written, want) {
			t.Errorf("the tree does not mention %s:\n%s", want, written)
		}
	}
}

// An empty program is still a tree, and showing it must not fail: a host asks for the tree
// before knowing whether there is anything in it.
func TestASTOfAnEmptyProgram(t *testing.T) {
	ast, _ := compile(t, "")

	out := bytes.NewBuffer(nil)
	if err := AST(out, ast); err != nil {
		t.Errorf("AST of an empty program: %v", err)
	}
	if out.Len() == 0 {
		t.Error("an empty program still has a tree to show")
	}
}

func TestInstructionsAreOnePerLine(t *testing.T) {
	_, program := compile(t, "1 + 2;\n")

	out := bytes.NewBuffer(nil)
	if err := Instructions(out, program.Instructions); err != nil {
		t.Fatalf("Instructions: %v", err)
	}

	lines := 0
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if strings.TrimSpace(line) != "" {
			lines++
		}
	}
	if lines != len(program.Instructions) {
		t.Errorf("wrote %d lines for %d instructions:\n%s", lines, len(program.Instructions), out)
	}
	if !strings.Contains(out.String(), "OpAdd") {
		t.Errorf("an addition should name its opcode:\n%s", out)
	}
}

// failing is a writer that cannot be written to, which is what a closed pipe is.
type failing struct{}

func (failing) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

// The error of a writer is returned, not swallowed. A host that pipes its output into
// something that has gone away has to hear about it — that is the whole reason this lives
// on the host side rather than printing from inside a phase.
func TestWriterErrorsAreReturned(t *testing.T) {
	ast, program := compile(t, "1 + 2;\n")

	if err := AST(failing{}, ast); err == nil {
		t.Error("AST swallowed the writer error")
	}
	if err := Instructions(failing{}, program.Instructions); err == nil {
		t.Error("Instructions swallowed the writer error")
	}
}
