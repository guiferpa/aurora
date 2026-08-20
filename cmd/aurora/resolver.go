package main

import (
	"os"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/resolver"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
)

// newResolver puts the front of the pipeline together for a resolver: reading a file, and
// turning what was read into a tree.
//
// The resolver knows neither. Source arrives through a port because a command line reads a
// disk and the playground reads a map it already holds, and a tree arrives through another
// because a phase does not know another phase — which leaves this, the only place allowed to
// know both.
func newResolver(tapeSize int, sourceRoot string) *resolver.Resolver {
	lx := lexer.New()
	ps := parser.New()

	return resolver.New(resolver.Options{
		SourceRoot: sourceRoot,
		Read:       os.ReadFile,
		Parse: func(filename string, id module.ID, source []byte, imports map[string]ast.Offer) (ast.AST, error) {
			tokens, err := lx.GetFilledTokens(source)
			if err != nil {
				return ast.AST{}, err
			}
			return ps.Parse(parser.ParseInput{
				Filename: filename,
				Tokens:   tokens,
				TapeSize: tapeSize,
				Module:   string(id),
				Imports:  imports,
			})
		},
		// What a file imports is read from the top of it, without a parse: a module has to be
		// read before whoever imports it, and knowing what to read cannot itself need one.
		Header: func(source []byte) ([]ast.UseDeclaration, error) {
			tokens, err := lx.GetFilledTokens(source)
			if err != nil {
				return nil, err
			}
			return parser.ScanUses(tokens), nil
		},
	})
}
