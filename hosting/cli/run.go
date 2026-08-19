package cli

import (
	"context"

	"github.com/guiferpa/aurora/byteutil"
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
	for _, each := range program.Ranges {
		if _, err := ev.EvaluateModule(program.Instructions, each.From, each.To, string(each.Module)); err != nil {
			return err
		}
	}
	return nil
}
