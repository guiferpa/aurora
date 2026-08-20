package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/resolver"
	"github.com/guiferpa/aurora/shared/printer"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
)

// A session is one file typed a line at a time, and nothing could read one back until Start
// took its writer. These go in the way a shell pipe does — lines in, everything the session
// said out — so what is pinned here is the REPL as someone uses it, not its parts.

// typed types the lines given and answers with everything the session wrote back. It puts the
// session together the way cmd/aurora does, since that is now where a session is made.
func typed(t *testing.T, lines string, tapeSize int) string {
	t.Helper()

	size := byteutil.TapeSize(tapeSize)
	out := &strings.Builder{}

	NewSession(NewSessionOptions{
		Lexer:   lexer.New(),
		Parser:  parser.New(),
		Emitter: emitter.New(emitter.NewEmitterOptions{TapeSize: size}),
		Evaluator: evaluator.New(evaluator.NewEvaluatorOptions{
			PrintBytes:   printer.Bytes(out, size),
			PrintChars:   printer.Chars(out, size),
			PrintDecimal: printer.Decimal(out, size),
			TapeSize:     size,
		}),
		In:       strings.NewReader(lines),
		Out:      out,
		TapeSize: size,
	}).Start()

	return out.String()
}

func TestSessionAnswersWithTheTape(t *testing.T) {
	cases := []struct {
		name  string
		lines string
		want  []string
	}{
		{name: "arithmetic", lines: "1 + 2;\n", want: []string{"= [0 0 0 0 0 0 0 3]"}},
		{
			// Text is a tape of its bytes, so this is what "hi" is, not a reading of it.
			name:  "text",
			lines: `"hi";` + "\n",
			want:  []string{"= [0 0 0 0 0 0 104 105]"},
		},
		{
			name:  "an identifier holds on the next line",
			lines: "ident x = 7;\nx * 2;\n",
			want:  []string{"= [0 0 0 0 0 0 0 14]"},
		},
		{
			// The declarations are held across lines: a struct declared on one line has to still
			// be known on the next, and a parser is built per line.
			name:  "a struct declared on one line is built on the next",
			lines: "struct Point { x, y };\nident p = Point{10, 20};\np.y;\n",
			want:  []string{"= [0 0 0 0 0 0 0 20]"},
		},
		{
			// The instructions of every line stay in one buffer, which is what keeps a defer's
			// range valid when it is called later.
			name:  "a defer written on one line is called on the next",
			lines: "ident double = defer { feed(0) * 2; };\ndouble(21);\n",
			want:  []string{"= [0 0 0 0 0 0 0 42]"},
		},
		{
			// One expression at a time, so each value is written where it happens.
			name:  "several expressions on one line",
			lines: "1; 2; 3;\n",
			want:  []string{"= [0 0 0 0 0 0 0 1]", "= [0 0 0 0 0 0 0 2]", "= [0 0 0 0 0 0 0 3]"},
		},
		{
			// What a program prints is its own writing, and it lands before the value of the
			// line that printed it. A print is worth what it showed — everything in Aurora is
			// worth something — so the session answers for it like for any other line.
			name:  "what the line printed comes before its value",
			lines: "printd 65;\n",
			want:  []string{"65", "= [0 0 0 0 0 0 0 65]"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := typed(t, tc.lines, 0)
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("session wrote %q, want it to contain %q", got, want)
				}
			}
		})
	}
}

// The width of a value is the session's, so a narrower tape wraps rather than growing.
func TestSessionHonoursTheTapeSize(t *testing.T) {
	got := typed(t, "255 + 1;\n", 1)

	if !strings.Contains(got, "= [0]") {
		t.Errorf("session wrote %q, want a one-byte tape holding zero", got)
	}
}

// A line that does not compile is answered and forgotten: the session is still there for the
// next one, which is most of what a REPL is for.
func TestSessionSurvivesALineThatFails(t *testing.T) {
	cases := []struct {
		name  string
		lines string
		fails string
	}{
		{name: "a line that does not parse", lines: "1 +;\n2;\n", fails: "unexpected token"},
		{name: "a name that was never set", lines: "nope;\n2;\n", fails: "identifier"},
		{name: "a division by zero", lines: "1 / 0;\n2;\n", fails: "divide by zero"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := typed(t, tc.lines, 0)

			if !strings.Contains(got, tc.fails) {
				t.Errorf("session wrote %q, want it to say %q", got, tc.fails)
			}
			if !strings.Contains(got, "= [0 0 0 0 0 0 0 2]") {
				t.Errorf("session wrote %q, want the line after the failure to have run", got)
			}
		})
	}
}

// A blank line is not an expression: it prompts again and says nothing.
func TestSessionSaysNothingForABlankLine(t *testing.T) {
	got := typed(t, "\n   \n", 0)

	if strings.Contains(got, "=") {
		t.Errorf("session wrote %q, want nothing but prompts", got)
	}
	if strings.Count(got, prompt) != 3 {
		t.Errorf("session wrote %d prompts, want one per line and one for the end", strings.Count(got, prompt))
	}
}

// typedIn is typed, in a project on disk: a use line reads a file, and where it reads from is
// where the session was started.
func typedIn(t *testing.T, dir, lines string, tapeSize int) string {
	t.Helper()
	t.Chdir(dir)

	size := byteutil.TapeSize(tapeSize)
	out := &strings.Builder{}
	lx := lexer.New()
	ps := parser.New()

	NewSession(NewSessionOptions{
		Lexer:   lx,
		Parser:  ps,
		Emitter: emitter.New(emitter.NewEmitterOptions{TapeSize: size}),
		Evaluator: evaluator.New(evaluator.NewEvaluatorOptions{
			PrintBytes:   printer.Bytes(out, size),
			PrintChars:   printer.Chars(out, size),
			PrintDecimal: printer.Decimal(out, size),
			TapeSize:     size,
		}),
		In:       strings.NewReader(lines),
		Out:      out,
		TapeSize: size,
		Resolver: resolver.New(resolver.Options{
			SourceRoot: "src",
			Read:       os.ReadFile,
			Parse: func(filename string, id module.ID, source []byte) (ast.AST, error) {
				tokens, err := lx.GetFilledTokens(source)
				if err != nil {
					return ast.AST{}, err
				}
				return ps.Parse(parser.ParseInput{
					Filename: filename,
					Tokens:   tokens,
					TapeSize: size,
					Module:   string(id),
				})
			},
		}),
	}).Start()

	return out.String()
}

// withModule writes a project with one module in it and answers where it is.
func withModule(t *testing.T, source string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "geometry.ar"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const geometry = "ident area = defer { feed(0) * feed(1); };\nident base = 10;"

// A use line brings the module in, and what is inside it answers on the lines after.
func TestASessionImports(t *testing.T) {
	got := typedIn(t, withModule(t, geometry), "use geometry as g;\nprintd g.area(3, 4);\nprintd g.base;\n", 0)

	for _, want := range []string{"12", "10"} {
		if !strings.Contains(got, want) {
			t.Errorf("the session said %q, want it to contain %q", got, want)
		}
	}
}

// A module is a program, so it runs once: importing it again is a use of what is already
// here, rather than binding its names a second time.
func TestAModuleIsBroughtInOnce(t *testing.T) {
	got := typedIn(t, withModule(t, "printd 1;\nident base = 10;"),
		"use geometry as g;\nuse geometry as h;\nprintd h.base;\n", 0)

	if strings.Count(got, "1\n") != 1 {
		t.Errorf("the module ran more than once: %q", got)
	}
	if strings.Contains(got, "conflict") {
		t.Errorf("importing twice bound its names twice: %q", got)
	}
	if !strings.Contains(got, "10") {
		t.Errorf("the second alias does not reach it: %q", got)
	}
}

// What is wrong is said when the line is read, not when it runs.
func TestASessionSaysWhatIsWrongWithAnImport(t *testing.T) {
	for _, tc := range []struct{ name, lines, want string }{
		{
			name:  "a module that is not there",
			lines: "use gone as g;\n",
			want:  "module gone is not there",
		},
		{
			name:  "a name the module does not have",
			lines: "use geometry as g;\nprintd g.volume;\n",
			want:  "module geometry has no volume",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := typedIn(t, withModule(t, geometry), tc.lines, 0)
			if !strings.Contains(got, tc.want) {
				t.Errorf("the session said %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

// A session with nowhere to read from takes no imports, and says so rather than accepting the
// line and doing nothing with it.
func TestASessionWithoutAResolverRefusesToPretend(t *testing.T) {
	got := typed(t, "use geometry as g;\nprintd g.base;\n", 0)

	if !strings.Contains(got, "nowhere to read a module from") {
		t.Errorf("the session said %q, want it to say it cannot import", got)
	}
}
