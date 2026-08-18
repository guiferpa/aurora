package cli

import (
	"context"
	"os"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/parser"
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

	tree, err := s.parser.Parse(parser.ParseInput{Filename: source, Tokens: tokens, TapeSize: s.tapeSize})
	if err != nil {
		return err
	}

	program, err := s.emitter.EmitProgram(tree)
	if err != nil {
		return err
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
