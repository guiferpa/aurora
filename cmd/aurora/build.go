package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/internal/cli"
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
	buildCmd.Flags().StringSliceP("loggers", "l", []string{}, "show what each phase produced (valid: lexer, parser, emitter)")
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

	loggers, err := cmd.Flags().GetStringSlice("loggers")
	if err != nil {
		return err
	}
	tapeSize, err := cmd.Flags().GetInt("tape-size")
	if err != nil {
		return err
	}

	_, err = cli.Build(cmd.Context(), cli.BuildInput{
		Source:     target.Source,
		OutputPath: output,
		Loggers:    loggers,
		Warnings:   os.Stderr,
		Stdout:     cmd.OutOrStdout(),
		TapeSize:   cli.ResolveTapeSize(tapeSize, target.TapeSize),
	})
	return err
}
