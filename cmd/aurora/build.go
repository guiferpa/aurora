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

func init() {
	buildCmd.Flags().StringSliceP("loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), builder)")
	buildCmd.Flags().StringP("output", "o", "", "output path for compiled binary (default: binary from aurora.toml)")
	buildCmd.Flags().StringP("source", "s", "", "custom source code to build")
	buildCmd.Flags().StringP("profile", "p", "main", "profile to build")
}

func runBuild(cmd *cobra.Command, args []string) error {
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
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	if output == "" {
		output = env.AbsPath(env.Profile.Binary)
	}
	loggers, err := cmd.Flags().GetStringSlice("loggers")
	if err != nil {
		return err
	}
	return cli.Build(cmd.Context(), cli.BuildInput{
		Source:     source,
		OutputPath: output,
		Loggers:    loggers,
	})
}
