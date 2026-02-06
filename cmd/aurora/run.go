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
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	env, err := cli.LoadEnviron("main")
	if err != nil {
		return err
	}
	source := env.AbsPath(env.Profile.Source)
	if len(args) > 0 {
		source = args[0]
	}
	var pl *evaluator.Player
	if player {
		pl = evaluator.NewPlayer(os.Stdin)
	}
	return cli.Run(cmd.Context(), cli.RunInput{
		Source:  source,
		Loggers: loggers,
		Stdin:   os.Stdin,
		Stdout:  ToMainWriter(),
		Player:  pl,
	})
}
