package cli

import (
	"context"
	"os"
	"path/filepath"
	"slices"

	"github.com/guiferpa/aurora/builder/evm"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

// BuildInput is the input for the Build handler.
type BuildInput struct {
	Entrypoint string   // path to .ar source
	OutputPath string   // path to write bytecode
	Loggers    []string // enabled loggers (lexer, parser, emitter, builder)
}

// Build compiles the Aurora source at Entrypoint and writes bytecode to OutputPath.
func Build(ctx context.Context, in BuildInput) error {
	bs, err := os.ReadFile(in.Entrypoint)
	if err != nil {
		return err
	}

	tokens, err := lexer.New(lexer.NewLexerOptions{
		EnableLogging: slices.Contains(in.Loggers, "lexer"),
	}).GetFilledTokens(bs)
	if err != nil {
		return err
	}

	ast, err := parser.New(tokens, parser.NewParserOptions{
		Filename:      in.Entrypoint,
		EnableLogging: slices.Contains(in.Loggers, "parser"),
	}).Parse()
	if err != nil {
		return err
	}

	insts, err := emitter.New(emitter.NewEmitterOptions{
		EnableLogging: slices.Contains(in.Loggers, "emitter"),
	}).Emit(ast)
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
