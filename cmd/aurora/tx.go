package main

import (
	"fmt"
	"math/big"

	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/hosting/cli"
)

var txCmd = &cobra.Command{
	Use:   "tx <function> [args...]",
	Short: "Send a transaction to a scope of a deployed contract",
	Long: `Send a transaction to a scope of a deployed contract.

A call is a question and a transaction is a change. "aurora call" asks a
contract something against the state as it is, costs nothing, and keeps nothing
it did on the way; "aurora tx" sends it, pays for it, and what it changed stays.

A scope that keeps something has to be sent. Asking it as a question answers,
and leaves the chain exactly as it was.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTx,
}

func init() {
	txCmd.Flags().Bool("pretend", false, "show what would be sent, and send nothing")
	txCmd.Flags().String("value", "0", "wei to carry with the transaction; nothing in Aurora sends ether back")
	txCmd.Flags().StringP("profile", "p", "main", "profile to send through")
}

func runTx(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("profile %s: rpc is required to send a transaction", profile)
	}
	if env.Profile.Privkey == "" {
		return fmt.Errorf("profile %s: privkey is required to send a transaction", profile)
	}
	deployed, ok := env.Manifest.Deploys[profile]
	if !ok {
		return fmt.Errorf("profile %s: no deploy found (run 'aurora deploy' first)", profile)
	}
	pretend, err := cmd.Flags().GetBool("pretend")
	if err != nil {
		return err
	}
	written, err := cmd.Flags().GetString("value")
	if err != nil {
		return err
	}
	// The same reading `aurora run --value` gets, so what a program is told off a chain and
	// what it is sent on one are one description.
	bytes, err := cli.ParseCallValue(written)
	if err != nil {
		return err
	}
	carried := new(big.Int).SetBytes(bytes)

	return cli.Tx(cmd.Context(), cli.TxInput{
		Function:        fn,
		ContractAddress: deployed.ContractAddress,
		RPC:             env.Profile.RPC,
		Privkey:         env.Profile.Privkey,
		Args:            args[1:],
		Pretend:         pretend,
		Value:           carried,
		Blocks:          blocksOfProfile(profile),
	})
}
