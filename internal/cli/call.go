package cli

import (
	"context"
	"fmt"
	"math/big"
	"strings"

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

func EncodeArgs(args []string) []byte {
	data := make([]byte, 0)
	for _, arg := range args {
		data = append(data, byteutil.Padding32Bytes([]byte(arg))...)
	}
	return data
}

// ParseArgs encodes each argument as a 32-byte ABI word, inferring type from the string:
// - bool: "true" / "false" (case-insensitive) → 0 or 1 right-padded to 32 bytes
// - number: decimal ("42") or hex ("0x2a") → uint256 big-endian
// - string: anything else; use "" for empty string; quoted strings have quotes stripped
func ParseArgs(args []string) []byte {
	data := make([]byte, 0)
	for _, arg := range args {
		data = append(data, parseArg(arg)...)
	}
	return data
}

func parseArg(arg string) []byte {
	// bool
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "true":
		return byteutil.Padding32Bytes([]byte{1})
	case "false":
		return byteutil.Padding32Bytes([]byte{0})
	}
	// number (decimal or 0x-prefixed hex)
	if n := parseNumber(arg); n != nil {
		b := make([]byte, 32)
		nb := n.Bytes()
		if len(nb) > 32 {
			copy(b, nb[len(nb)-32:])
		} else {
			copy(b[32-len(nb):], nb)
		}
		return b
	}
	// string (strip surrounding double quotes; "" → empty)
	s := strings.TrimSpace(arg)
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) && len(s) >= 2 {
		s = s[1 : len(s)-1]
	}
	return byteutil.Padding32Bytes([]byte(s))
}

func parseNumber(s string) *big.Int {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n := new(big.Int)
		if _, ok := n.SetString(s[2:], 16); !ok {
			return nil
		}
		return n
	}
	n := new(big.Int)
	if _, ok := n.SetString(s, 10); !ok {
		return nil
	}
	return n
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
