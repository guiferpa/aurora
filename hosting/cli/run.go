package cli

import (
	"context"
	"os"
	"slices"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/shared/trace"
)

// Run compiles the source and evaluates it.
//
// The unit of compilation is the file. Aurora had a namespace layer that treated a directory
// as the unit and compiled every .ar file next to the entry point, which made independent
// programs in one folder collide; it was removed until the module system is designed (see
// docs/module_system_design.md).
func (s *Session) Run(ctx context.Context, source string) error {
	if err := byteutil.ValidateTapeSize(s.tapeSize); err != nil {
		return err
	}

	bs, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	tokens, err := s.lexer.GetFilledTokens(bs)
	if err != nil {
		return err
	}
	if slices.Contains(s.loggers, "lexer") {
		if err := trace.Tokens(s.stdout, tokens); err != nil {
			return err
		}
	}

	tree, err := s.parser.Parse(parser.ParseInput{Filename: source, Tokens: tokens})
	if err != nil {
		return err
	}
	// A phase returns what it made and does not show it; showing is decided here, once the
	// phase has finished, which is why the output is no longer on-time.
	if slices.Contains(s.loggers, "parser") {
		if err := trace.AST(s.stdout, tree); err != nil {
			return err
		}
	}

	program, err := s.emitter.EmitProgram(tree)
	if err != nil {
		return err
	}
	if slices.Contains(s.loggers, "emitter") {
		if err := trace.Instructions(s.stdout, program.Instructions); err != nil {
			return err
		}
	}

	// A warning is something worth knowing before running, not a reason to refuse the source.
	ReportWarnings(s.warnings, source, program.Warnings)

	ev, err := s.evaluator()
	if err != nil {
		return err
	}
	if _, err := ev.Evaluate(program.Instructions); err != nil {
		return err
	}
	return nil
}
