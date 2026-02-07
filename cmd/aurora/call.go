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

func runCall(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: call <function> [arg0 [arg1 ...]]")
	}
	fn := args[0]
	profileName := "main"
	callArgs := args[1:]
	env, err := cli.LoadEnviron(profileName)
	if err != nil {
		return err
	}
	if env.Profile.RPC == "" {
		return fmt.Errorf("profile %s: rpc is required for call", profileName)
	}
	contractAddr := ""
	if env.Manifest.Deploys != nil {
		if d, ok := env.Manifest.Deploys[profileName]; ok {
			contractAddr = d.ContractAddress
		}
	}
	if contractAddr == "" {
		return fmt.Errorf("profile %s: no deploy found (run 'aurora deploy' first)", profileName)
	}
	return cli.Call(cmd.Context(), cli.CallInput{
		Function:        fn,
		ContractAddress: contractAddr,
		RPC:             env.Profile.RPC,
		Args:            callArgs,
	})
}
