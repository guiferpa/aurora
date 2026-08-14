package main

import (
	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/logger"
)

// Whether a manifest is needed is decided by each command, not here: "aurora run x.ar"
// compiles a file and has no business demanding that a project exist around it, while
// deploy and call read rpc and privkey from a profile and do require one.
var rootCmd = &cobra.Command{
	Use: "aurora",
	// Cobra prints the error and then the whole usage block, and CommandError prints the
	// error again in colour — three blocks for one failure. Both are silenced so a failed
	// command says one thing. Usage is what "aurora help" and "--help" are for.
	SilenceErrors: true,
	SilenceUsage:  true,
}

func main() {
	rootCmd.AddCommand(versionCmd, runCmd, replCmd, buildCmd, deployCmd, callCmd, initCmd)
	logger.CommandError(rootCmd.Execute())
}
