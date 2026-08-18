package cli

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fatih/color"

	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/eval"
	"github.com/guiferpa/aurora/wire/ir"
)

// TestExtension marks a test file. A test belongs to the source file of the same name:
// greeting.test.ar tests greeting.ar, which is evaluated first so the test can see what it
// declared. There is no module system yet, so this is how a test reaches the code it
// checks; when there is one, the rule becomes "a test belongs to the module of the same
// name", which is the same sentence.
const TestExtension = ".test.ar"

// FileReport is what happened in one test file.
type FileReport struct {
	Path    string
	Results []eval.AssertResult
	Err     error // the file could not be compiled or run at all
}

// Passed reports whether every assertion in the file held and nothing went wrong.
func (r FileReport) Passed() bool {
	if r.Err != nil {
		return false
	}
	for _, result := range r.Results {
		if !result.Passed {
			return false
		}
	}
	return true
}

// TestReport is what happened across every file.
type TestReport struct {
	Files  []FileReport
	Passed int
	Failed int
}

// brokenFiles counts the files that could not be compiled or run at all.
func (r TestReport) brokenFiles() int {
	broken := 0
	for _, file := range r.Files {
		if file.Err != nil {
			broken++
		}
	}
	return broken
}

// OK reports whether the run as a whole succeeded.
func (r TestReport) OK() bool {
	if r.Failed > 0 {
		return false
	}
	for _, file := range r.Files {
		if file.Err != nil {
			return false
		}
	}
	return true
}

// Test runs the test files it is given and writes a report.
//
// Which files those are is settled before the session exists: a test file names its own
// project, and the width it is compiled at comes from there — see TestFiles.
func (s *Session) Test(ctx context.Context, files []string) (TestReport, error) {
	if err := ValidateTapeSize(s.tapeSize); err != nil {
		return TestReport{}, err
	}
	if len(files) == 0 {
		return TestReport{}, fmt.Errorf("no %s files found", TestExtension)
	}

	report := TestReport{Files: make([]FileReport, 0, len(files))}
	for _, path := range files {
		file := s.runTestFile(path)
		for _, result := range file.Results {
			if result.Passed {
				report.Passed++
			} else {
				report.Failed++
			}
		}
		report.Files = append(report.Files, file)
	}

	writeReport(s.stdout, report)
	return report, nil
}

// TestFiles resolves a target into the files to run and the width to compile them at.
//
// With no target it uses the main profile; with a name, that profile; with a path ending in
// .test.ar, only that file. A profile is searched from the directory of its source, down to
// the leaves — nothing above it.
//
// It answers before a session exists because the width it finds is what the phases are built
// with.
func TestFiles(target string, tapeSize int) ([]string, int, error) {
	if strings.HasSuffix(target, TestExtension) {
		// A test file named directly is still a file of its project, and it has to run at
		// the width the source it tests was written in.
		project, err := ProjectTapeSize(target)
		if err != nil {
			return nil, 0, err
		}
		return []string{target}, ResolveTapeSize(tapeSize, project), nil
	}

	resolved, err := ResolveTarget(target)
	if err != nil {
		return nil, 0, err
	}
	files, err := findTests(filepath.Dir(resolved.Source))
	if err != nil {
		return nil, 0, err
	}
	return files, ResolveTapeSize(tapeSize, resolved.TapeSize), nil
}

// findTests walks root and its subdirectories for test files. Only what is at or below the
// starting point is considered.
func findTests(root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), TestExtension) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}

// failedAt names the file a failure came from, and only when it is the source: a failure in
// the test file is reported against the test being run, which is what the reader is looking at.
func failedAt(file, source string, err error) error {
	if file != source {
		return err
	}
	return fmt.Errorf("%s: %w", filepath.Base(source), err)
}

// SourceForTest returns the file a test belongs to: greeting.test.ar -> greeting.ar.
func SourceForTest(path string) string {
	return strings.TrimSuffix(path, TestExtension) + SourceExtension
}

// runTestFile evaluates a test together with the source it belongs to.
//
// Both run in one evaluator, so what the source declares is in scope for the test — its
// bindings and its deferred scopes alike, since the defer counter belongs to the environ.
// Whatever the source does at its top level happens too, prints included.
func (s *Session) runTestFile(path string) FileReport {
	report := FileReport{Path: path}

	// A test belongs to the source file of the same name, and cannot run without it: it
	// would not see the code it is meant to check.
	source := SourceForTest(path)
	if _, err := os.Stat(source); err != nil {
		report.Err = fmt.Errorf("no %s next to it: a test needs the source file it belongs to",
			filepath.Base(source))
		return report
	}

	// The two files are one scope, so they are one set of declarations: a struct declared in
	// the source is known in the test. They are parsed apart because each keeps its own name —
	// which is what decides that assert belongs to the test file — and its own line numbers.
	declarations := parser.NewDeclarations()

	// Both programs go into one instruction stream, and each is run as a range of it. A
	// deferred scope records where its body sits as indexes into the stream, so handing
	// the evaluator a different slice afterwards would point those indexes at unrelated
	// instructions — a call would then run whatever happened to be there. The REPL keeps
	// one buffer across lines for the same reason. Each range clears the temps first,
	// since the two were emitted separately and both number their labels from zero.
	instructions := make([]ir.Instruction, 0)
	boundary := 0

	for _, file := range []string{source, path} {
		bs, err := os.ReadFile(file)
		if err != nil {
			report.Err = err
			return report
		}

		tokens, err := s.lexer.GetFilledTokens(bs)
		if err != nil {
			report.Err = failedAt(file, source, err)
			return report
		}

		tree, err := s.parser.Parse(parser.ParseInput{
			Filename:     file,
			Tokens:       tokens,
			Declarations: declarations,
		})
		if err != nil {
			report.Err = failedAt(file, source, err)
			return report
		}

		program, err := s.emitter.EmitProgram(tree)
		if err != nil {
			report.Err = failedAt(file, source, err)
			return report
		}

		boundary = len(instructions)
		instructions = append(instructions, program.Instructions...)
	}

	ev, err := s.evaluator()
	if err != nil {
		report.Err = err
		return report
	}

	if _, err := ev.EvaluateRange(instructions, 0, uint64(boundary)); err != nil {
		report.Err = fmt.Errorf("%s: %w", source, err)
		return report
	}
	if _, err := ev.EvaluateRange(instructions, uint64(boundary), uint64(len(instructions))); err != nil {
		report.Err = err
		return report
	}

	report.Results = ev.GetAssertResults()
	return report
}

// displayPath shows a path relative to where the command was run, which is how someone
// reading the report thinks about their own files. A profile resolves its source against
// the project root, so the paths arrive absolute.
func displayPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	relative, err := filepath.Rel(cwd, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return path
	}
	return relative
}

func writeReport(w io.Writer, report TestReport) {
	if w == nil {
		return
	}

	pass := color.New(color.FgGreen).SprintFunc()
	fail := color.New(color.FgRed).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	for _, file := range report.Files {
		_, _ = fmt.Fprintln(w, displayPath(file.Path))
		if file.Err != nil {
			_, _ = fmt.Fprintf(w, "  %s  %s\n", fail("ERROR"), file.Err)
			continue
		}
		if len(file.Results) == 0 {
			_, _ = fmt.Fprintf(w, "  %s\n", dim("no assertions"))
		}
		for _, result := range file.Results {
			if result.Passed {
				_, _ = fmt.Fprintf(w, "  %s    %s\n", pass("ok"), result.Message)
				continue
			}
			_, _ = fmt.Fprintf(w, "  %s  %s\n", fail("FAIL"), result.Message)
		}
		_, _ = fmt.Fprintln(w)
	}

	summary := fmt.Sprintf("%d passed, %d failed in %d file", report.Passed, report.Failed, len(report.Files))
	if len(report.Files) != 1 {
		summary += "s"
	}
	if broken := report.brokenFiles(); broken > 0 {
		summary += fmt.Sprintf(", %d could not run", broken)
	}
	if report.OK() {
		_, _ = fmt.Fprintln(w, pass(summary))
		return
	}
	_, _ = fmt.Fprintln(w, fail(summary))
}
