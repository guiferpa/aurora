package cli

import (
	"context"
	"os"
	"path/filepath"
	"slices"

	"github.com/guiferpa/aurora/builder/evm"
	"github.com/guiferpa/aurora/linker"
)

// BuildInput is the input for the Build handler.
type BuildInput struct {
	Source     string   // path to .ar source
	OutputPath string   // path to write bytecode
	Loggers    []string // enabled loggers (lexer, parser, emitter, builder)
}

// Build compiles the Aurora source at Source and writes bytecode to OutputPath.
func Build(ctx context.Context, in BuildInput) error {
	l, err := linker.NewLinker(linker.NewLinkerOptions{
		Source:  in.Source,
		Loggers: in.Loggers,
	})
	if err != nil {
		return err
	}
	insts, err := l.Resolve()
	if err != nil {
		return err
	}

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
			insts,
			evm.NewBuilderOptions{
				EnableLogging: slices.Contains(in.Loggers, "builder"),
			},
		).Build(fd)
		return err
	}()
}
