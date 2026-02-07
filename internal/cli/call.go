package cli

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const ABI_WORD_BYTES = 32

// CallInput is the input for the Call handler.
type CallInput struct {
	Function        string   // function name (selector = Keccak256(function)[:4])
	ContractAddress string
	RPC             string
	Args            []string // optional arguments (decimal or 0x-prefixed hex), ABI-encoded as uint256 each
}

// encodeCallData returns selector + args as ABI-encoded (each arg 32 bytes, big-endian).
func encodeCallData(selector []byte, args []string) ([]byte, error) {
	data := append([]byte(nil), selector...)
	for _, a := range args {
		a = strings.TrimSpace(a)
		var n *big.Int
		if strings.HasPrefix(a, "0x") || strings.HasPrefix(a, "0X") {
			a = strings.TrimPrefix(strings.TrimPrefix(a, "0x"), "0X")
			b, err := hex.DecodeString(a)
			if err != nil {
				return nil, fmt.Errorf("invalid hex arg %q: %w", a, err)
			}
			n = new(big.Int).SetBytes(b)
		} else {
			var ok bool
			n, ok = new(big.Int).SetString(a, 10)
			if !ok {
				return nil, fmt.Errorf("invalid number arg %q", a)
			}
		}
		word := make([]byte, ABI_WORD_BYTES)
		b := n.Bytes()
		copy(word[ABI_WORD_BYTES-len(b):], b)
		data = append(data, word...)
	}
	return data, nil
}

// Call performs an eth_call and prints the result.
func Call(ctx context.Context, in CallInput) error {
	selector := crypto.Keccak256([]byte(in.Function))[:4]
	data, err := encodeCallData(selector, in.Args)
	if err != nil {
		return err
	}
	contract := common.HexToAddress(in.ContractAddress)

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
