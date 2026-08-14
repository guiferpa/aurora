package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/fatih/color"

	"github.com/guiferpa/aurora/builder/evm"
	"github.com/guiferpa/aurora/byteutil"
)

// BuildInput is the input for the Build handler.
type BuildInput struct {
	Source     string    // path to .ar source
	OutputPath string    // path to write bytecode
	Loggers    []string  // enabled loggers (lexer, parser, emitter, builder)
	TapeSize   int       // width in bytes of every value; zero means the default
	Warnings   io.Writer // receives compiler warnings; nil discards them
	Stdout     io.Writer // receives the report; nil says nothing
}

// BuildReport is what a build produced.
type BuildReport struct {
	Source       string // the file that was compiled
	Binary       string // where the bytecode landed
	Instructions int    // how many instructions the emitter produced
	Bytes        int    // how large the bytecode is
	TapeSize     int    // width in bytes of every value in it
}

// Build compiles the Aurora source at Source and writes bytecode to OutputPath.
//
// It reports what it produced: a build used to be silent on success, which left no way to
// tell a build that wrote a binary from one that found nothing to do, and no sign of where
// the binary went — the path may come from a profile rather than from the command line.
func Build(ctx context.Context, in BuildInput) (BuildReport, error) {
	report := BuildReport{
		Source:   in.Source,
		Binary:   in.OutputPath,
		TapeSize: byteutil.TapeSize(in.TapeSize),
	}

	if err := ValidateTapeSize(in.TapeSize); err != nil {
		return report, err
	}
	program, err := Compile(in.Source, in.TapeSize, in.Loggers)
	if err != nil {
		return report, err
	}
	ReportWarnings(in.Warnings, in.Source, program.Warnings)
	report.Instructions = len(program.Instructions)

	if err := os.MkdirAll(filepath.Dir(in.OutputPath), 0o755); err != nil {
		return report, err
	}
	fd, err := os.Create(in.OutputPath)
	if err != nil {
		return report, err
	}

	err = func() (err error) {
		defer func() {
			closeErr := fd.Close()
			if closeErr != nil && err == nil {
				// return closeErr only if there was no build error
				err = closeErr
			}
		}()
		report.Bytes, err = evm.NewBuilder(
			program.Instructions,
			evm.NewBuilderOptions{
				EnableLogging: slices.Contains(in.Loggers, "builder"),
				TapeSize:      in.TapeSize,
			},
		).Build(fd)
		return err
	}()
	if err != nil {
		return report, err
	}

	writeBuildReport(in.Stdout, report)
	return report, nil
}

// writeBuildReport says where the binary went and what it is made of: the shape of the
// program above the line, its size on disk below it.
func writeBuildReport(w io.Writer, report BuildReport) {
	if w == nil {
		return
	}

	bold := color.New(color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	_, _ = fmt.Fprintf(w, "✨ %s → %s\n", displayPath(report.Source), bold(displayPath(report.Binary)))
	_, _ = fmt.Fprintf(w, "   %s\n", dim(fmt.Sprintf("%s, %s, %d-byte tapes",
		plural(report.Instructions, "instruction"),
		plural(report.Bytes, "byte"),
		report.TapeSize,
	)))
}

func plural(count int, noun string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, noun)
	}
	return fmt.Sprintf("%d %ss", count, noun)
}
