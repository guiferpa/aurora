package cli

import (
	"context"

	"github.com/guiferpa/aurora/byteutil"

	"github.com/guiferpa/aurora/wire/ir"
)

// Run compiles the source and everything it imports, and evaluates it.
//
// The unit of compilation is the module, and a module is a file: what the entry says it needs
// is found, loaded once each, and run before it — the order the resolver answered in. A file
// importing nothing is a program of one module, which is what every program used to be.
func (s *Session) Run(ctx context.Context, source string) error {
	if err := byteutil.ValidateTapeSize(s.tapeSize); err != nil {
		return err
	}

	program, err := s.compile(source)
	if err != nil {
		return err
	}

	// A warning is something worth knowing before running, not a reason to refuse the source.
	s.report(program)

	ev, err := s.evaluator()
	if err != nil {
		return err
	}
	for at, each := range program.Ranges {
		if _, err := ev.EvaluateBlocks(program.Blocks, ir.Point{Block: each.Top}, program.StopsAt(at), string(each.Module)); err != nil {
			return err
		}
	}
	return nil
}
