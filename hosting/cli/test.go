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

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/loader"
	"github.com/guiferpa/aurora/wire/eval"
)

// TestExtension marks a test file, and marking it is all it does: a test file is a program
// like any other, and this is how `aurora test` knows which files to run and how the parser
// knows which files may hold an assertion.
//
// It used to mean more. A test belonged to the source file of the same name and the two ran
// as one scope, because there was no other way for a test to reach the code it checks. Now a
// test names what it needs with `use`, like everything else does.
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
	if err := byteutil.ValidateTapeSize(s.tapeSize); err != nil {
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

// runRanges runs every range in order, and the last one is the test file itself.
//
// A failure before it comes from a module the test named, which is code being checked rather
// than the check, so it says which file it came from. A failure in the test is reported
// against the test being run, which is what the reader is already looking at.
func runRanges(ev *evaluator.Evaluator, program loader.Program) error {
	for i, each := range program.Ranges {
		_, err := ev.EvaluateBlocks(program.Blocks, each.Top, program.StopsAt(i), string(each.Module))
		if err == nil {
			continue
		}
		if i < len(program.Ranges)-1 {
			return fmt.Errorf("%s: %w", each.Filename, err)
		}
		return err
	}
	return nil
}

// runTestFile runs one test file and collects what its assertions said.
//
// It is compiled and run exactly as `aurora run` would compile and run it, because that is
// what it is: a program that names what it needs, whose modules load once each and run before
// it. The only thing this does that running does not is read the results afterwards.
func (s *Session) runTestFile(path string) FileReport {
	report := FileReport{Path: path}

	program, err := s.compile(path)
	if err != nil {
		report.Err = err
		return report
	}

	ev, err := s.evaluator()
	if err != nil {
		report.Err = err
		return report
	}

	if err := runRanges(ev, program); err != nil {
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
