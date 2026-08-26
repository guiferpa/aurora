package main

import (
	"fmt"
	"os"

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

// The exit code is a contract with whoever runs the CLI — a shell, a CI job — so it is
// decided here, where the process is, and nowhere else. Diagnostics go to stderr, so a
// pipeline reading a program's output does not swallow them.
func main() {
	rootCmd.AddCommand(versionCmd, runCmd, testCmd, replCmd, buildCmd, deployCmd, callCmd, txCmd, initCmd, installStdCmd)

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprint(os.Stderr, logger.CommandError(err))
		os.Exit(2)
	}
}
