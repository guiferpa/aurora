package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/internal/cli"
)

var callCmd = &cobra.Command{
	Use:   "call <function> [profile]",
	Short: "Call program on a blockchain",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runCall,
}

func init() {
	callCmd.Flags().Bool("pretend", false, "pretend/simulate the call (dry run)")
	callCmd.Flags().StringP("profile", "p", "main", "profile to call")
}

func runCall(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: call <function> [arg0 [arg1 ...]]")
	}
	fn := args[0]
	profile, err := cmd.Flags().GetString("profile")
	if err != nil {
		return err
	}
	env, err := cli.LoadEnviron(profile)
	if err != nil {
		return err
	}
	if env.Profile.RPC == "" {
		return fmt.Errorf("profile %s: rpc is required for call", profile)
	}
	if len(env.Manifest.Deploys) < 1 {
		return fmt.Errorf("no deploys found (run 'aurora deploy' first)")
	}
	d, ok := env.Manifest.Deploys[profile]
	if !ok {
		return fmt.Errorf("profile %s: no deploy found (run 'aurora deploy' first)", profile)
	}
	pretend, err := cmd.Flags().GetBool("pretend")
	if err != nil {
		return err
	}
	return cli.Call(cmd.Context(), cli.CallInput{
		Function:        fn,
		ContractAddress: d.ContractAddress,
		RPC:             env.Profile.RPC,
		Args:            args[1:],
		Pretend:         pretend,
	})
}
