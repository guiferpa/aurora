package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/internal/cli"
	"github.com/guiferpa/aurora/manifest"
)

var (
	deployMinTipGwei   int
	deployMinMaxFeeGwei int
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy program to a blockchain (uses binary, rpc, and privkey from aurora.toml)",
	Args:  cobra.NoArgs,
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().IntVar(&deployMinTipGwei, "min-tip", 0, "minimum priority fee in Gwei (overrides default when RPC suggests too low)")
	deployCmd.Flags().IntVar(&deployMinMaxFeeGwei, "min-max-fee", 0, "minimum max fee per gas in Gwei (overrides default when RPC suggests too low)")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	profileName := "main"
	env, err := cli.LoadEnviron(profileName)
	if err != nil {
		return err
	}
	if env.Profile.RPC == "" {
		return fmt.Errorf("profile %s: rpc is required for deploy", profileName)
	}
	if env.Profile.Privkey == "" {
		return fmt.Errorf("profile %s: privkey is required for deploy", profileName)
	}
	address, deployTxHash, deployedAt, err := cli.Deploy(cmd.Context(), cli.DeployInput{
		BinaryPath:     env.AbsPath(env.Profile.Binary),
		RPC:            env.Profile.RPC,
		Privkey:        env.Profile.Privkey,
		MinTipGwei:     deployMinTipGwei,
		MinMaxFeeGwei:  deployMinMaxFeeGwei,
	})
	if err != nil {
		return err
	}
	return manifest.PersistDeploy(env.Root, profileName, address, deployTxHash, deployedAt.Format(time.RFC3339))
}
