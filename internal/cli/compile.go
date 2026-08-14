package cli

import (
	"os"
	"slices"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

// Compile turns a source file into a program: read, lex, parse, emit.
//
// The program carries any warnings the compiler wants to raise. They do not stop it: a
// warning is something worth knowing before running, not a reason to refuse the source.
//
// The unit of compilation is the file. Aurora had a namespace layer that treated a
// directory as the unit and compiled every .ar file next to the entry point, which made
// independent programs in one folder collide; it was removed until the module system is
// designed (see docs/module_system_design.md).
func Compile(source string, tapeSize int, loggers []string) (emitter.Program, error) {
	bs, err := os.ReadFile(source)
	if err != nil {
		return emitter.Program{}, err
	}

	tokens, err := lexer.New(lexer.NewLexerOptions{
		EnableLogging: slices.Contains(loggers, "lexer"),
	}).GetFilledTokens(bs)
	if err != nil {
		return emitter.Program{}, err
	}

	ast, err := parser.New(parser.NewParserOptions{
		Filename:      source,
		Tokens:        tokens,
		EnableLogging: slices.Contains(loggers, "parser"),
		TapeSize:      tapeSize,
	}).Parse()
	if err != nil {
		return emitter.Program{}, err
	}

	return emitter.New(emitter.NewEmitterOptions{
		EnableLogging: slices.Contains(loggers, "emitter"),
		TapeSize:      tapeSize,
	}).EmitProgram(ast)
}
