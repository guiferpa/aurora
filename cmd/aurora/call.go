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
	fn := args[0]
	profileName := "main"
	if len(args) > 1 {
		profileName = args[1]
	}
	env, err := cli.LoadEnviron(profileName)
	if err != nil {
		return err
	}
	if env.Profile.RPC == "" {
		return fmt.Errorf("profile %s: rpc is required for call", profileName)
	}
	if env.Profile.ContractAddress == "" {
		return fmt.Errorf("profile %s: contract_address is required for call", profileName)
	}
	return cli.Call(cmd.Context(), cli.CallInput{
		Function:        fn,
		ContractAddress: env.Profile.ContractAddress,
		RPC:             env.Profile.RPC,
	})
}
