package cli

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

// Sending a transaction to a scope of a deployed contract.
//
// It is a command of its own rather than a flag on call, and the two words mean what they mean
// everywhere else a chain is spoken to: a call is a question, answered against the state as it
// is and costing nothing; a transaction is a change, mined, paid for, and kept.
//
// Nothing decides between them from what a program looks like. What the compiler knows is used
// to refuse the wrong one — a scope that writes, called, is a write thrown away — and never to
// spend somebody's money because a name looked like it wanted it.

// TxGasLimit is what one call to a scope is allowed to burn. It is well under the deploy's,
// because a scope is a jump inside a contract and not a contract being written.
const TxGasLimit = 500_000

// TxInput is what sending a transaction to a scope takes.
type TxInput struct {
	Function        string
	ContractAddress string
	RPC             string
	Privkey         string
	Args            []string
	Pretend         bool
	MinTipGwei      int
	MinMaxFeeGwei   int
	// Out is where it says what it is doing. Nil is stdout, which is what a command wants and
	// what a test never does.
	Out io.Writer
	// Blocks is the program the contract was built from, when whoever asked could compile it.
	// A scope that changes nothing is still sent — there are reasons to want a receipt — and
	// is worth a word, because gas was spent on an answer a call gives free.
	Blocks []ir.Block
}

// Tx sends a transaction to one scope of a deployed contract, and waits for it to be mined.
//
// It waits rather than answering the hash and leaving, because what a transaction did is not
// known until it is mined: a reverted one is a transaction too, and somebody who was told
// "sent" and nothing else has to go and find out.
func Tx(ctx context.Context, in TxInput) error {
	out := in.Out
	if out == nil {
		out = os.Stdout
	}

	if writes, found := ScopeWrites(WritesInput{Blocks: in.Blocks, Function: in.Function}); found && !writes {
		sayLine(out, warnATxThatWritesNothing(in.Function))
	}

	data := append(EncodeSelector(in.Function), ParseArgs(in.Args)...)
	contract := common.HexToAddress(in.ContractAddress)

	if in.Pretend {
		say(out, "Contract:   0x%x (%d bytes)\n", contract, len(contract.Bytes()))
		say(out, "Function:   %s\n", in.Function)
		say(out, "Data:       %s\n", byteutil.ToHexPretty(data))
		sayLine(out, "Nothing was sent: this is what the transaction would carry.")
		return nil
	}

	privateKey, err := readPrivkey(in.Privkey)
	if err != nil {
		return err
	}
	client, err := ethclient.Dial(in.RPC)
	if err != nil {
		return err
	}

	signed, err := signedCall(ctx, client, privateKey, contract, data, in.MinTipGwei, in.MinMaxFeeGwei)
	if err != nil {
		return err
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return err
	}

	sayLine(out, "Calling:", in.Function)
	sayLine(out, "Sent by:", crypto.PubkeyToAddress(privateKey.PublicKey).Hex())
	sayLine(out, "TX:", signed.Hash().Hex())

	receipt, err := waitForReceipt(ctx, client, signed.Hash(), out)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("the transaction reverted: %s kept nothing, and the gas was spent — check %s on an explorer",
			in.Function, signed.Hash().Hex())
	}

	sayLine(out, "Gas used:", receipt.GasUsed)
	sayLine(out, "Mined in block:", receipt.BlockNumber)
	return nil
}

// signedCall builds and signs a transaction to an address, with the fees the network asks for
// and the floors a testnet needs to include one at all.
func signedCall(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, to common.Address, data []byte, minTip, minFee int) (*types.Transaction, error) {
	from := crypto.PubkeyToAddress(key.PublicKey)

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, err
	}
	tip, fee, err := suggestFees(ctx, client, minTip, minFee)
	if err != nil {
		return nil, err
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: fee,
		Gas:       TxGasLimit,
		To:        &to,
		Value:     big.NewInt(0),
		Data:      data,
	})
	return types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
}

// waitForReceipt polls until the transaction is mined, or until whoever asked gives up.
func waitForReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash, out io.Writer) (*types.Receipt, error) {
	started := time.Now()
	deadline := started.Add(RECEIPT_POLL_TIMEOUT)

	for time.Now().Before(deadline) {
		if receipt, err := client.TransactionReceipt(ctx, hash); err == nil && receipt != nil {
			elapsed := time.Since(started)
			say(out, "\rWaiting for confirmation... %dm %ds   \n", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
			return receipt, nil
		}
		elapsed := time.Since(started)
		say(out, "\rWaiting for confirmation... %dm %ds   ", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
		select {
		case <-ctx.Done():
			say(out, "\n")
			return nil, ctx.Err()
		case <-time.After(RECEIPT_POLL_INTERVAL):
		}
	}

	say(out, "\n")
	return nil, fmt.Errorf("not mined within %v, and it may still be pending: check %s on an explorer",
		RECEIPT_POLL_TIMEOUT, hash.Hex())
}

// say writes what a command is doing, and does not fail over it.
//
// Every other write in this package is checked, and this one is not, on purpose: what these
// two commands print is progress while money moves. A transaction that was sent and mined was
// sent and mined whether or not the line saying so reached a terminal, and turning a broken
// pipe into an error would report the wrong thing happening.
func say(out io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(out, format, args...)
}

// sayLine is say for something with nothing to fill in.
func sayLine(out io.Writer, args ...any) {
	_, _ = fmt.Fprintln(out, args...)
}
