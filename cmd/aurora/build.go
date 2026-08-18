package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/hosting/cli"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

var buildCmd = &cobra.Command{
	Use:   "build [profile | file.ar]",
	Short: "Build binary from source code",
	Long: `Build binary from source code.

With no argument, the "main" profile from aurora.toml is used. A name selects
another profile. A path ending in .ar builds that file, with no manifest
involved:

  aurora build                  the "main" profile
  aurora build dev              the "dev" profile
  aurora build src/main.ar      that file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBuild,
}

func init() {
	buildCmd.Flags().StringP("output", "o", "", "output path for compiled binary (default: binary from aurora.toml, or the file name without extension)")
	buildCmd.Flags().IntP("tape-size", "t", 0, "bytes per value (1-32, default 8; overrides tape_size from aurora.toml)")
}

func runBuild(cmd *cobra.Command, args []string) error {
	var arg string
	if len(args) > 0 {
		arg = args[0]
	}
	target, err := cli.ResolveTarget(arg)
	if err != nil {
		return err
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	if output == "" {
		// A profile carries its own output path; a loose file has none, so the binary
		// lands next to where the command runs, named after the source.
		output = target.Binary
		if output == "" {
			output = cli.DefaultBinaryPath(target.Source)
		}
	}

	tapeSize, err := cmd.Flags().GetInt("tape-size")
	if err != nil {
		return err
	}

	size := cli.ResolveTapeSize(tapeSize, target.TapeSize)

	// A build compiles and writes bytecode; nothing evaluates, so no evaluator is made.
	_, err = cli.NewSession(cli.NewSessionOptions{
		Lexer:    lexer.New(lexer.NewLexerOptions{}),
		Parser:   parser.New(parser.NewParserOptions{}),
		Emitter:  emitter.New(emitter.NewEmitterOptions{TapeSize: size}),
		TapeSize: size,
		Stdout:   cmd.OutOrStdout(),
		Warnings: os.Stderr,
	}).Build(cmd.Context(), target.Source, output)
	return err
}
