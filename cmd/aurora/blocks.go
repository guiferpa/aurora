package main

import (
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/hosting/cli"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

// blocksOfProfile compiles what a profile points at, for the two commands that need to know
// what a scope does before they speak to a chain.
//
// It answers nothing rather than an error when the source cannot be compiled, and that is on
// purpose: a contract was deployed from a source that may since have changed, or been moved,
// or belong to somebody else's checkout. Not knowing is a reason to say less, never a reason
// to stop somebody reaching a contract that is already out there.
func blocksOfProfile(profile string) []ir.Block {
	env, err := cli.LoadEnviron(profile)
	if err != nil {
		return nil
	}
	target, err := cli.ResolveTarget(profile)
	if err != nil {
		return nil
	}
	size := cli.ResolveTapeSize(0, target.TapeSize)

	blocks, err := cli.NewSession(cli.NewSessionOptions{
		Lexer:    lexer.New(),
		Parser:   parser.New(),
		Emitter:  emitter.New(emitter.NewEmitterOptions{TapeSize: size}),
		Resolver: newResolver(size, target.SourceRoot),
		TapeSize: size,
	}).Blocks(env.AbsPath(env.Profile.Source))
	if err != nil {
		return nil
	}
	return blocks
}
