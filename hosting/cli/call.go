package cli

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/guiferpa/aurora/byteutil"
)

// CallInput is the input for the Call handler.
type CallInput struct {
	Function        string // function name (selector = Keccak256(function))
	ContractAddress string
	RPC             string
	Args            []string // optional arguments (decimal or 0x-prefixed hex), ABI-encoded as uint256 each
	Pretend         bool
}

func EncodeSelector(selector string) []byte {
	return byteutil.Padding32Bytes(crypto.Keccak256([]byte(selector)))
}

// Call performs an eth_call and prints the result.
func Call(ctx context.Context, in CallInput) error {
	selector := EncodeSelector(in.Function)
	args := ParseArgs(in.Args)
	contract := common.HexToAddress(in.ContractAddress)
	data := append(selector, args...)

	if in.Pretend {
		fmt.Printf("Contract:   0x%x (%d bytes)\n", contract, len(contract.Bytes()))
		fmt.Printf("Function:   0x%x (%d bytes)\n", selector, len(selector))
		fmt.Printf("Arguments:  0x%x (%d bytes)\n", args, len(args))
		fmt.Printf("Data:       %s\n", byteutil.ToHexPretty(data))
		return nil
	}

	client, err := ethclient.Dial(in.RPC)
	if err != nil {
		return err
	}

	msg := ethereum.CallMsg{
		To:   &contract,
		Data: data,
	}

	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Result: %v\n", result)
	return nil
}
