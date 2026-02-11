package main

import (
	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/internal/cli"
	"github.com/guiferpa/aurora/logger"
)

var rootCmd = &cobra.Command{
	Use: "aurora",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		switch args[0] {
		case "init", "version", "help", "repl":
			return nil
		}
		return cli.RequireManifest()
	},
}

func main() {
	rootCmd.AddCommand(versionCmd, runCmd, replCmd, buildCmd, deployCmd, callCmd, initCmd)
	logger.CommandError(rootCmd.Execute())
}
