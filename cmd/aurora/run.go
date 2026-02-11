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
	runCmd.Flags().BoolP("player", "r", false, "enable player mode (stdin)")
	runCmd.Flags().StringP("source", "s", "", "custom source code to run")
	runCmd.Flags().StringP("profile", "p", "main", "profile to run")
}

func runRun(cmd *cobra.Command, args []string) error {
	profile, err := cmd.Flags().GetString("profile")
	if err != nil {
		return err
	}
	env, err := cli.LoadEnviron(profile)
	if err != nil {
		return err
	}
	source, err := cmd.Flags().GetString("source")
	if err != nil {
		return err
	}
	if source == "" {
		source = env.AbsPath(env.Profile.Source)
	}
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
