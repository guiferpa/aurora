package cli

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// CallInput is the input for the Call handler.
type CallInput struct {
	Function        string // function name (selector = Keccak256(function)[:4])
	ContractAddress string
	RPC             string
}

// Call performs an eth_call and prints the result.
func Call(ctx context.Context, in CallInput) error {
	selector := crypto.Keccak256([]byte(in.Function))[:4]
	contract := common.HexToAddress(in.ContractAddress)

	client, err := ethclient.Dial(in.RPC)
	if err != nil {
		return err
	}

	msg := ethereum.CallMsg{
		To:   &contract,
		Data: selector,
	}

	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Result: %v\n", result)
	return nil
}
