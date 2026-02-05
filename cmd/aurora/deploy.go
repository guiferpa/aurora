package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/internal/cli"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy program to a blockchain (reads target, rpc_url, private_key_path from manifest)",
	Args:  cobra.NoArgs,
	RunE:  runDeploy,
}

func runDeploy(cmd *cobra.Command, args []string) error {
	env, err := cli.LoadEnviron("main")
	if err != nil {
		return err
	}
	if env.Profile.RPCURL == "" {
		return fmt.Errorf("profile main: rpc_url is required for deploy")
	}
	if env.Profile.PrivateKeyPath == "" {
		return fmt.Errorf("profile main: private_key_path is required for deploy")
	}
	return cli.Deploy(cmd.Context(), cli.DeployInput{
		BytecodePath:   env.AbsPath(env.Profile.Target),
		RPCURL:         env.Profile.RPCURL,
		PrivateKeyPath: env.AbsPath(env.Profile.PrivateKeyPath),
	})
}
