package cli

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// DeployInput is the input for the Deploy handler.
type DeployInput struct {
	BinaryPath string // path to compiled binary
	RPC       string
	Privkey   string // path to file containing hex private key
}

// Deploy sends the bytecode to the chain and returns the contract address and deploy timestamp. The caller should persist these via manifest.PersistDeploy.
func Deploy(ctx context.Context, in DeployInput) (address string, deployedAt time.Time, err error) {
	bs, err := os.ReadFile(in.BinaryPath)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("read binary from %s: %w", in.BinaryPath, err)
	}

	keyHex, err := os.ReadFile(in.Privkey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("read private key from %s: %w", in.Privkey, err)
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimSpace(string(keyHex)))
	if err != nil {
		return "", time.Time{}, err
	}
	from := crypto.PubkeyToAddress(privateKey.PublicKey)

	client, err := ethclient.Dial(in.RPC)
	if err != nil {
		return "", time.Time{}, err
	}

	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", time.Time{}, err
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", time.Time{}, err
	}

	tx := types.NewContractCreation(nonce, big.NewInt(0), 3_000_000, gasPrice, bs)

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", time.Time{}, err
	}
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return "", time.Time{}, err
	}

	log.Println("Deploy TX:", signedTx.Hash().Hex())

	contractAddr := crypto.CreateAddress(from, nonce)
	deployedAt = time.Now().UTC()
	fmt.Println("Contract deployed at:", contractAddr.Hex())
	return contractAddr.Hex(), deployedAt, nil
}
