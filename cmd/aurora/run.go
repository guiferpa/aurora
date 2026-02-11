package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/internal/cli"
)

var runCmd = &cobra.Command{
	Use:   "run [file]",
	Short: "Run program directly from source code",
	RunE:  runRun,
}

func init() {
	runCmd.Flags().StringSliceP("loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), evaluator)")
	runCmd.Flags().BoolP("player", "p", false, "enable player mode (stdin)")
}

func runRun(cmd *cobra.Command, args []string) error {
	env, err := cli.LoadEnviron("main")
	if err != nil {
		return err
	}
	source := env.AbsPath(env.Profile.Source)
	var pl *evaluator.Player
	if player, _ := cmd.Flags().GetBool("player"); player {
		pl = evaluator.NewPlayer(os.Stdin)
	}
	loggers, err := cmd.Flags().GetStringSlice("loggers")
	if err != nil {
		return err
	}
	return cli.Run(cmd.Context(), cli.RunInput{
		Source:  source,
		Loggers: loggers,
		Stdin:   os.Stdin,
		Stdout:  ToMainWriter(),
		Player:  pl,
		Args:    args,
	})
}
