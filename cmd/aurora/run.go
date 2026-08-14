package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/internal/cli"
)

var runCmd = &cobra.Command{
	Use:   "run [profile | file.ar] [args...]",
	Short: "Run program directly from source code",
	Long: `Run program directly from source code.

With no argument, the "main" profile from aurora.toml is used. A name selects
another profile. A path ending in .ar runs that file, with no manifest
involved:

  aurora run                  the "main" profile
  aurora run dev              the "dev" profile
  aurora run examples/x.ar    that file

Anything after the target is passed to the program and read with feed(n).`,
	RunE: runRun,
}

func init() {
	runCmd.Flags().StringSliceP("loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), evaluator)")
	runCmd.Flags().BoolP("player", "r", false, "enable player mode (stdin)")
	runCmd.Flags().IntP("tape-size", "t", 0, "bytes per value (1-32, default 8; overrides tape_size from aurora.toml)")
}

func runRun(cmd *cobra.Command, args []string) error {
	var arg string
	var programArgs []string
	if len(args) > 0 {
		arg, programArgs = args[0], args[1:]
	}
	target, err := cli.ResolveTarget(arg)
	if err != nil {
		return err
	}

	var pl *evaluator.Player
	if player, _ := cmd.Flags().GetBool("player"); player {
		pl = evaluator.NewPlayer(os.Stdin)
	}
	loggers, err := cmd.Flags().GetStringSlice("loggers")
	if err != nil {
		return err
	}
	tapeSize, err := cmd.Flags().GetInt("tape-size")
	if err != nil {
		return err
	}

	return cli.Run(cmd.Context(), cli.RunInput{
		Source:   target.Source,
		Loggers:  loggers,
		Stdin:    os.Stdin,
		Stdout:   ToMainWriter(),
		Warnings: os.Stderr,
		EchoOut:  ToEchoWriter(),
		Player:   pl,
		Args:     programArgs,
		TapeSize: cli.ResolveTapeSize(tapeSize, target.TapeSize),
	})
}
