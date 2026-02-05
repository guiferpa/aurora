package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/internal/cli"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy program to a blockchain",
	Args:  cobra.NoArgs,
	RunE:  runDeploy,
}

func runDeploy(cmd *cobra.Command, args []string) error {
	env, err := cli.LoadEnviron("main")
	if err != nil {
		return err
	}
	if env.Profile.RPC == "" {
		return fmt.Errorf("profile main: rpc is required for deploy")
	}
	if env.Profile.Privkey == "" {
		return fmt.Errorf("profile main: privkey is required for deploy")
	}
	return cli.Deploy(cmd.Context(), cli.DeployInput{
		BinaryPath: env.AbsPath(env.Profile.Target),
		RPC:        env.Profile.RPC,
		Privkey:    env.AbsPath(env.Profile.Privkey),
	})
}
