package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/guiferpa/aurora/builder/evm"
)

// BuildInput is the input for the Build handler.
type BuildInput struct {
	Source     string    // path to .ar source
	OutputPath string    // path to write bytecode
	Loggers    []string  // enabled loggers (lexer, parser, emitter, builder)
	TapeSize   int       // width in bytes of every value; zero means the default
	Warnings   io.Writer // receives compiler warnings; nil discards them
}

// Build compiles the Aurora source at Source and writes bytecode to OutputPath.
func Build(ctx context.Context, in BuildInput) error {
	if err := ValidateTapeSize(in.TapeSize); err != nil {
		return err
	}
	program, err := Compile(in.Source, in.TapeSize, in.Loggers)
	if err != nil {
		return err
	}
	ReportWarnings(in.Warnings, program.Warnings)

	if err := os.MkdirAll(filepath.Dir(in.OutputPath), 0o755); err != nil {
		return err
	}
	fd, err := os.Create(in.OutputPath)
	if err != nil {
		return err
	}

	return func() (err error) {
		defer func() {
			closeErr := fd.Close()
			if closeErr != nil && err == nil {
				// return closeErr only if there was no build error
				err = closeErr
			}
		}()
		_, err = evm.NewBuilder(
			program.Instructions,
			evm.NewBuilderOptions{
				EnableLogging: slices.Contains(in.Loggers, "builder"),
				TapeSize:      in.TapeSize,
			},
		).Build(fd)
		return err
	}()
}
