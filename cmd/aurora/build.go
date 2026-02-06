package main

import (
	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/internal/cli"
)

var buildCmd = &cobra.Command{
	Use:   "build [file]",
	Short: "Build binary from source code",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runBuild,
}

func runBuild(cmd *cobra.Command, args []string) error {
	env, err := cli.LoadEnviron("main")
	if err != nil {
		return err
	}
	source := env.AbsPath(env.Profile.Source)
	if len(args) > 0 {
		source = args[0]
	}
	outPath := output
	if outPath == "" {
		outPath = env.AbsPath(env.Profile.Binary)
	}
	return cli.Build(cmd.Context(), cli.BuildInput{
		Source:    source,
		OutputPath: outPath,
		Loggers:   loggers,
	})
}
