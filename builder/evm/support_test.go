package evm

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

// instructionsOf builds the instruction stream by opcode, which is all Warnings reads.
func instructionsOf(opcodes ...byte) []ir.Instruction {
	insts := make([]ir.Instruction, 0, len(opcodes))
	for i, op := range opcodes {
		insts = append(insts, ir.NewInstruction(byteutil.FromUint64(uint64(i)), op, ir.Nothing(), ir.Nothing()))
	}
	return insts
}

func TestWarningsNamesWhatDoesNotReachTheBytecode(t *testing.T) {
	cases := []struct {
		name     string
		opcodes  []byte
		want     []string
		wantNone bool
	}{
		{
			// What a contract is made of today: values, names, arguments, arithmetic and a
			// return, inside a scope that becomes a dispatcher entry.
			name:     "a program the builder writes whole",
			opcodes:  []byte{ir.OpDefer, ir.OpBeginScope, ir.OpGetFeed, ir.OpSave, ir.OpAdd, ir.OpReturn, ir.OpIdent},
			wantNone: true,
		},
		{
			name:    "a branch",
			opcodes: []byte{ir.OpIf, ir.OpJump},
			// Two instructions, one feature, one thing to say about it.
			want: []string{"if does not reach the bytecode yet"},
		},
		{
			name:    "a comparison",
			opcodes: []byte{ir.OpBigger},
			want:    []string{"a comparison does not reach the bytecode yet"},
		},
		{
			name:    "a tape operation",
			opcodes: []byte{ir.OpPull, ir.OpHead},
			want:    []string{"a tape operation does not reach the bytecode yet"},
		},
		{
			name:    "a shape",
			opcodes: []byte{ir.OpJoin, ir.OpField},
			want:    []string{"shape does not reach the bytecode yet"},
		},
		{
			name:    "calling a scope",
			opcodes: []byte{ir.OpPushFeed, ir.OpCall},
			want:    []string{"calling a scope does not reach the bytecode yet"},
		},
		{
			// A log is not a gap: it is absent on purpose, and the wording says so.
			name:    "a print",
			opcodes: []byte{ir.OpPrintDecimal},
			want:    []string{"printd writes a log", "by decision"},
		},
		{
			name:    "an assertion",
			opcodes: []byte{ir.OpAssert},
			want:    []string{"assert belongs to 'aurora test'", "by decision"},
		},
		{
			name:    "each print speaks for itself",
			opcodes: []byte{ir.OpPrintBytes, ir.OpPrintChars, ir.OpPrintDecimal},
			want:    []string{"printb", "printc", "printd"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			warnings := Warnings(instructionsOf(tc.opcodes...))

			if tc.wantNone {
				if len(warnings) != 0 {
					t.Errorf("said %v about a program it writes whole", warnings)
				}
				return
			}

			said := make([]string, 0, len(warnings))
			for _, warning := range warnings {
				said = append(said, warning.Message)
			}
			whole := strings.Join(said, "\n")

			for _, want := range tc.want {
				if !strings.Contains(whole, want) {
					t.Errorf("said %q, want it to mention %q", whole, want)
				}
			}
		})
	}
}

// The same feature used twice is one thing to say, not two.
func TestWarningsSayEachThingOnce(t *testing.T) {
	warnings := Warnings(instructionsOf(
		ir.OpIf, ir.OpJump, ir.OpIf, ir.OpJump, ir.OpBigger, ir.OpSmaller,
	))

	if len(warnings) != 2 {
		t.Errorf("said %d things about two features: %v", len(warnings), warnings)
	}
}

// They arrive in the order the program uses them, so the first thing a reader is told about
// is the first thing that goes missing.
func TestWarningsFollowTheProgram(t *testing.T) {
	warnings := Warnings(instructionsOf(ir.OpPrintDecimal, ir.OpIf))

	if len(warnings) != 2 {
		t.Fatalf("said %d things, want two", len(warnings))
	}
	if !strings.Contains(warnings[0].Message, "printd") {
		t.Errorf("said %q first, want the print", warnings[0].Message)
	}
}

// A warning that names a feature and not a place leaves the person to search for it. The IR
// carries where every instruction came from now, so the backend points at the first line that
// used what it cannot carry.
func TestWarningsPointAtWhereTheFeatureWasUsed(t *testing.T) {
	const source = `ident sum = defer { feed(0) + feed(1); };
printb sum(1, 2);`

	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	tree, err := parser.New().Parse(parser.ParseInput{Filename: "main.ar", Tokens: tokens})
	if err != nil {
		t.Fatalf("parser: %v", err)
	}
	insts, err := emitter.New(emitter.NewEmitterOptions{}).Emit(tree)
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}

	for _, warning := range Warnings(insts) {
		if !strings.Contains(warning.Message, "calling a scope") {
			continue
		}
		if !warning.Positioned() {
			t.Fatal("the warning about calling a scope has no place to point at")
		}
		if warning.Line != 2 {
			t.Errorf("it points at line %d, want the line the call was written on", warning.Line)
		}
		return
	}
	t.Fatal("nothing warned about calling a scope")
}

// Every feature the backend cannot carry names the line it was written on. A warning that
// names only the feature leaves the person to search, which is what this used to do.
func TestEveryPendingFeatureNamesItsPlace(t *testing.T) {
	cases := []struct {
		name   string
		source string
		says   string
		line   int
	}{
		{name: "an if", source: "printb 1;\nprintb if true { 1; };", says: "if", line: 2},
		{name: "a call", source: "ident f = defer { 1; };\nprintb f();", says: "calling a scope", line: 2},
		{name: "a comparison", source: "printb 1;\nprintb 2 bigger 1;", says: "a comparison", line: 2},
		{name: "and or or", source: "printb 1;\nprintb true and false;", says: "and/or", line: 2},
		{name: "an exponent", source: "printb 1;\nprintb 2 ^ 3;", says: "^", line: 2},
		// A tape literal is itself a tape operation — it builds the run byte by byte — so
		// the first place the program uses one is the literal, not the pull below it.
		{name: "a tape operation", source: "printb 1;\nident t = [1];\nprintb pull t 2;", says: "a tape operation", line: 2},
		{name: "a shape", source: "shape Point { x, y };\nprintb Point{1, 2}.x;", says: "shape", line: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := lexer.New().GetFilledTokens([]byte(tc.source))
			if err != nil {
				t.Fatalf("lexer: %v", err)
			}
			tree, err := parser.New().Parse(parser.ParseInput{Filename: "main.ar", Tokens: tokens})
			if err != nil {
				t.Fatalf("parser: %v", err)
			}
			insts, err := emitter.New(emitter.NewEmitterOptions{}).Emit(tree)
			if err != nil {
				t.Fatalf("emitter: %v", err)
			}

			for _, warning := range Warnings(insts) {
				if !strings.Contains(warning.Message, tc.says) {
					continue
				}
				if warning.Line != tc.line {
					t.Errorf("%q points at line %d, want %d", warning.Message, warning.Line, tc.line)
				}
				return
			}
			t.Fatalf("nothing warned about %q", tc.says)
		})
	}
}
