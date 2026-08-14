package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/internal/cli"
)

var testCmd = &cobra.Command{
	Use:   "test [profile | file.test.ar]",
	Short: "Run the test files of a project",
	Long: `Run the test files of a project.

A test belongs to the source file of the same name: greeting.test.ar tests
greeting.ar, which runs first so the test can see what it declared.

With no argument, the "main" profile is used and its source directory is
searched, down to the leaves. A name selects another profile. A path runs that
file alone:

  aurora test                      the "main" profile
  aurora test dev                  the "dev" profile
  aurora test src/greeting.test.ar that file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTest,
}

func init() {
	testCmd.Flags().StringSliceP("loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter, evaluator)")
	testCmd.Flags().IntP("tape-size", "t", 0, "bytes per value (1-32, default 8; overrides tape_size from aurora.toml)")
}

func runTest(cmd *cobra.Command, args []string) error {
	var target string
	if len(args) > 0 {
		target = args[0]
	}
	loggers, err := cmd.Flags().GetStringSlice("loggers")
	if err != nil {
		return err
	}
	tapeSize, err := cmd.Flags().GetInt("tape-size")
	if err != nil {
		return err
	}

	report, err := cli.Test(cmd.Context(), cli.TestInput{
		Target:   target,
		Stdout:   os.Stdout,
		TapeSize: tapeSize,
		Loggers:  loggers,
	})
	if err != nil {
		return err
	}
	if !report.OK() {
		// The report has already been written; this is only what the exit code carries, so
		// that a script or a CI job can tell what happened.
		if report.Failed > 0 {
			return fmt.Errorf("%d of %d assertions failed", report.Failed, report.Passed+report.Failed)
		}
		return fmt.Errorf("some test files could not run")
	}
	return nil
}
