package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/hosting/cli"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Start an Aurora project in the current directory",
	Long: `Start an Aurora project in the current directory.

Writes the manifest and the layout it describes: aurora.toml, a program in
src/main.ar and its tests in src/main.test.ar. A file that already exists is
left alone.`,
	Args: cobra.NoArgs,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	return cli.Init(cli.InitInput{
		Dir:         dir,
		ProjectName: filepath.Base(dir),
		Stdout:      os.Stdout,
	})
}
