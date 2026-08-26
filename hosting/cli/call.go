package cli

import (
	"context"
	"io"
	"os"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

// CallInput is the input for the Call handler.
type CallInput struct {
	Function        string // function name (selector = Keccak256(function))
	ContractAddress string
	RPC             string
	Args            []string // optional arguments (decimal or 0x-prefixed hex), ABI-encoded as uint256 each
	Pretend         bool
	// Blocks is the program the contract was built from, when whoever asked could compile it.
	// It is what says whether this scope changes anything, and a call over one that does is
	// refused rather than answered — the answer would look right and the chain would be
	// exactly as it was.
	//
	// Empty is a call with no program in hand, which asks the question and says nothing.
	Blocks []ir.Block
	// Out is where it says what it is doing. Nil is stdout.
	Out io.Writer
}

func EncodeSelector(selector string) []byte {
	return byteutil.Padding32Bytes(crypto.Keccak256([]byte(selector)))
}

// Call asks a question of a deployed contract, against the state as it is, and prints the
// answer. Nothing it does is kept.
//
// A scope that changes something is refused rather than asked. It happened on a real chain
// before this was here: a scope that read a counter, added one and kept it answered 1 every
// time, because every call started from the state the last one did not change.
func Call(ctx context.Context, in CallInput) error {
	if writes, found := ScopeWrites(WritesInput{Blocks: in.Blocks, Function: in.Function}); found && writes {
		return refuseACallThatWrites(in.Function)
	}

	out := in.Out
	if out == nil {
		out = os.Stdout
	}

	selector := EncodeSelector(in.Function)
	args := ParseArgs(in.Args)
	contract := common.HexToAddress(in.ContractAddress)
	data := append(selector, args...)

	if in.Pretend {
		say(out, "Contract:   0x%x (%d bytes)\n", contract, len(contract.Bytes()))
		say(out, "Function:   0x%x (%d bytes)\n", selector, len(selector))
		say(out, "Arguments:  0x%x (%d bytes)\n", args, len(args))
		say(out, "Data:       %s\n", byteutil.ToHexPretty(data))
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

	say(out, "Result: %v\n", result)
	return nil
}
