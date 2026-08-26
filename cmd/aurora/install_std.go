package main

import (
	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/hosting/cli"
	"github.com/guiferpa/aurora/stdlib"
)

var installStdCmd = &cobra.Command{
	Use:   "install-std",
	Short: "Write the standard library where the language reads it from",
	Long: `Write the standard library where the language reads it from.

A module named std/... is read from $AURORA_ROOT/lib, or $HOME/.aurora/lib when
AURORA_ROOT says nothing. This writes it there, out of this binary, so a
toolchain that was downloaded is complete without a git checkout.

The files stay on disk and are ordinary Aurora: open them, read them, patch one
if you have a reason. Nothing here reads them from anywhere else.`,
	Args: cobra.NoArgs,
	RunE: runInstallStd,
}

func init() {
	installStdCmd.Flags().Bool("force", false, "write over a standard library that is already there")
}

func runInstallStd(cmd *cobra.Command, args []string) error {
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	return cli.InstallStd(cli.InstallStdInput{
		Files: stdlib.Files(),
		Force: force,
		Out:   cmd.OutOrStdout(),
	})
}
