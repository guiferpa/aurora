package cli

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// exampleSources lists every example that runs on its own, the project ones included.
//
// A test file belongs to the source of the same name and needs it in scope, so it runs under
// TestExamplesTestsPass rather than by itself.
func exampleSources(t *testing.T) ([]string, error) {
	t.Helper()
	root := filepath.Join(repoRoot(t), "examples")

	sources := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), SourceExtension) || strings.HasSuffix(d.Name(), TestExtension) {
			return nil
		}
		sources = append(sources, path)
		return nil
	})
	return sources, err
}

// Every example must run. The header of each file documents the output it produces, and
// that output was pasted from a real run — this keeps them from rotting.
func TestExamplesRun(t *testing.T) {
	sources, err := exampleSources(t)
	if err != nil {
		t.Fatalf("walking examples: %v", err)
	}
	if len(sources) == 0 {
		t.Fatal("no examples found")
	}

	for _, source := range sources {
		t.Run(filepath.Base(source), func(t *testing.T) {
			err := Run(t.Context(), RunInput{
				Source: source,
				Stdout: io.Discard,
			})
			if err != nil {
				t.Errorf("%s failed: %v", source, err)
			}
		})
	}
}

// The examples that are tests must pass, which also covers the pairing rule end to end.
func TestExamplesTestsPass(t *testing.T) {
	report, err := Test(t.Context(), TestInput{
		Target: filepath.Join(repoRoot(t), "examples", "greeting.test.ar"),
		Stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("running the example tests: %v", err)
	}
	if !report.OK() {
		t.Errorf("the example tests should pass: %d failed", report.Failed)
	}
	if report.Passed == 0 {
		t.Error("expected assertions to have run")
	}
}

// declaredOutput reads the "#- Output:" block an example opens with: the lines it says it
// prints. Examples that document two tape widths side by side use a different header and
// answer nil, since a single run cannot be compared against both.
func declaredOutput(t *testing.T, source string) []string {
	t.Helper()

	bs, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("reading %s: %v", source, err)
	}

	lines := strings.Split(string(bs), "\n")
	start := slices.Index(lines, "#- Output:")
	if start < 0 {
		return nil
	}

	declared := make([]string, 0)
	for _, line := range lines[start+1:] {
		body, ok := strings.CutPrefix(line, "#-   ")
		if !ok {
			break
		}
		declared = append(declared, body)
	}
	return declared
}

// Every example says what it prints, and that block is pasted from a real run. This is what
// keeps the promise: the header is compared against the output, so an example cannot drift
// from the language without the suite noticing.
//
// It caught three of them the day text stopped being a reel, which is exactly the drift it
// exists for — the checking was done by hand then, and by hand it does not last.
func TestExamplesMatchTheirDeclaredOutput(t *testing.T) {
	// The whole tree, not only the top of it: the sources inside examples/project declare
	// what they print like every other example, and nothing was comparing them — the header
	// of one was checked by running it in a terminal, which is how the last three rotted.
	sources, err := exampleSources(t)
	if err != nil {
		t.Fatalf("walking examples: %v", err)
	}

	checked := 0
	for _, source := range sources {
		name, err := filepath.Rel(repoRoot(t), source)
		if err != nil {
			name = source
		}

		declared := declaredOutput(t, source)
		if len(declared) == 0 {
			continue
		}
		checked++

		t.Run(name, func(t *testing.T) {
			stdout := &strings.Builder{}
			if err := Run(t.Context(), RunInput{Source: source, Stdout: stdout}); err != nil {
				t.Fatalf("%s failed: %v", source, err)
			}

			printed := make([]string, 0)
			for _, line := range strings.Split(stdout.String(), "\n") {
				if strings.TrimSpace(line) != "" {
					printed = append(printed, line)
				}
			}

			if strings.Join(printed, "\n") != strings.Join(declared, "\n") {
				t.Errorf("the header of %s no longer matches what it prints:\ndeclared:\n  %s\nprinted:\n  %s",
					name, strings.Join(declared, "\n  "), strings.Join(printed, "\n  "))
			}
		})
	}

	if checked == 0 {
		t.Fatal("no example declared an output block, so nothing was compared")
	}
}
