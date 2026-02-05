package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/internal/cli"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create an aurora.toml manifest in the current directory",
	Args:  cobra.NoArgs,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	return cli.Init(cli.InitInput{
		Dir:         dir,
		ProjectName: filepath.Base(dir),
	})
}
