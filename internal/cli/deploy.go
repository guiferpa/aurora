package cli

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	MIN_BYTECODE_LEN      = 12             // init code (CODECOPY + RETURN) is at least 12 bytes
	DEPLOY_GAS_LIMIT      = 3_000_000
	RECEIPT_POLL_INTERVAL = 3 * time.Second
	RECEIPT_POLL_TIMEOUT  = 10 * time.Minute // allow time for congested networks and indexer delay

	// Small floor so we never send 0 (some RPCs return 0 on testnets). Real fee usually comes from SuggestGasTipCap/SuggestGasPrice.
	MIN_GAS_TIP_CAP_GWEI = 1 // 1 Gwei floor for priority fee
	MIN_GAS_FEE_CAP_GWEI = 2 // 2 Gwei floor for max fee (base + tip on low-activity testnets)
)

// DeployInput is the input for the Deploy handler.
type DeployInput struct {
	BinaryPath   string // path to compiled binary (raw bytes or 0x-prefixed hex)
	RPC          string
	Privkey      string // private key in hex (no 0x prefix)
	MinTipGwei   int    // min priority fee in Gwei (0 = use default)
	MinMaxFeeGwei int   // min max fee per gas in Gwei (0 = use default)
}

// decodeBytecode reads raw bytes from the file. If content starts with "0x" or "0X", it is decoded as hex; otherwise it is used as raw bytecode.
func decodeBytecode(raw []byte) ([]byte, error) {
	trimmed := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(trimmed, "0x") && !strings.HasPrefix(trimmed, "0X") {
		return raw, nil
	}
	trimmed = trimmed[2:]
	trimmed = strings.ReplaceAll(trimmed, "\n", "")
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	if len(trimmed)%2 != 0 {
		return nil, fmt.Errorf("hex string has odd length")
	}
	decoded, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("decode hex bytecode: %w", err)
	}
	return decoded, nil
}

// Deploy sends the bytecode to the chain and returns the contract address, the deploy transaction hash, and the deploy timestamp. The caller should persist these via manifest.PersistDeploy.
func Deploy(ctx context.Context, in DeployInput) (address string, deployTxHash string, deployedAt time.Time, err error) {
	raw, err := os.ReadFile(in.BinaryPath)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("read binary from %s: %w", in.BinaryPath, err)
	}

	bs, err := decodeBytecode(raw)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("bytecode from %s: %w", in.BinaryPath, err)
	}
	if len(bs) < MIN_BYTECODE_LEN {
		return "", "", time.Time{}, fmt.Errorf("bytecode too short (%d bytes); need at least %d", len(bs), MIN_BYTECODE_LEN)
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimSpace(in.Privkey))
	if err != nil {
		return "", "", time.Time{}, err
	}
	from := crypto.PubkeyToAddress(privateKey.PublicKey)

	client, err := ethclient.Dial(in.RPC)
	if err != nil {
		return "", "", time.Time{}, err
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return "", "", time.Time{}, err
	}

	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", "", time.Time{}, err
	}

	// EIP-1559 (DynamicFeeTx) for Sepolia and other modern chains.
	// Use minimum fees so the tx is included (allow override via DeployInput for bad RPCs).
	gwei := big.NewInt(1e9)
	minTipGwei := MIN_GAS_TIP_CAP_GWEI
	if in.MinTipGwei > 0 {
		minTipGwei = in.MinTipGwei
	}
	minFeeGwei := MIN_GAS_FEE_CAP_GWEI
	if in.MinMaxFeeGwei > 0 {
		minFeeGwei = in.MinMaxFeeGwei
	}
	minTipCap := new(big.Int).Mul(big.NewInt(int64(minTipGwei)), gwei)
	minFeeCap := new(big.Int).Mul(big.NewInt(int64(minFeeGwei)), gwei)

	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("suggest gas tip: %w", err)
	}
	if gasTipCap == nil || gasTipCap.Cmp(minTipCap) < 0 {
		gasTipCap = minTipCap
	}

	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("latest block: %w", err)
	}
	baseFee := head.BaseFee
	if baseFee == nil {
		baseFee = big.NewInt(0)
	}
	// maxFeePerGas = baseFee*2 + tip, but at least minFeeCap so tx is included on testnets
	gasFeeCap := new(big.Int).Mul(baseFee, big.NewInt(2))
	gasFeeCap.Add(gasFeeCap, gasTipCap)
	if gasFeeCap.Cmp(minFeeCap) < 0 {
		gasFeeCap = minFeeCap
	}

	// Log gas fees so user can verify on explorer (helps debug "pending forever").
	tipGwei := new(big.Int).Div(gasTipCap, gwei)
	feeCapGwei := new(big.Int).Div(gasFeeCap, gwei)
	fmt.Printf("Gas: tip=%s Gwei, maxFee=%s Gwei (min %d/%d Gwei for inclusion)\n", tipGwei.String(), feeCapGwei.String(), minTipGwei, minFeeGwei)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       DEPLOY_GAS_LIMIT,
		To:        nil, // contract creation
		Value:     big.NewInt(0),
		Data:      bs,
	})

	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		return "", "", time.Time{}, err
	}
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return "", "", time.Time{}, err
	}

	txHash := signedTx.Hash()
	fmt.Println("Binary deployed:", in.BinaryPath)
	fmt.Println("Bytecode size:", len(bs), "bytes")
	fmt.Println("Contract deployed by:", from.Hex())
	fmt.Println("Deploy TX:", txHash.Hex())

	// Wait for receipt and ensure tx succeeded (contract actually deployed).
	deadline := time.Now().Add(RECEIPT_POLL_TIMEOUT)
	waitStart := time.Now()
	for time.Now().Before(deadline) {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil && receipt != nil {
			elapsed := time.Since(waitStart)
			fmt.Printf("\rWaiting for confirmation... %dm %ds   \n", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
			if receipt.Status != types.ReceiptStatusSuccessful {
				return "", "", time.Time{}, fmt.Errorf("deploy tx reverted (status %d); check tx %s on explorer", receipt.Status, txHash.Hex())
			}
			contractAddr := receipt.ContractAddress.Hex()
			fmt.Println("Contract deployed at:", contractAddr)
			deployedAt = time.Now().UTC()
			fmt.Println("Deployed at:", deployedAt.Format(time.RFC3339))
			return contractAddr, txHash.Hex(), deployedAt, nil
		}
		elapsed := time.Since(waitStart)
		fmt.Printf("\rWaiting for confirmation... %dm %ds   ", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
		select {
		case <-ctx.Done():
			fmt.Print("\n")
			return "", "", time.Time{}, ctx.Err()
		case <-time.After(RECEIPT_POLL_INTERVAL):
			// poll again
		}
	}

	fmt.Print("\n")
	// Tx was broadcast but not mined in time (e.g. network/indexer delay). Tell user where to look once it confirms.
	expectedContract := crypto.CreateAddress(from, nonce)
	return "", "", time.Time{}, fmt.Errorf(
		"deploy tx not mined within %v (tx may still be pending). Check tx %s on explorer; contract will be at %s once confirmed",
		RECEIPT_POLL_TIMEOUT, txHash.Hex(), expectedContract.Hex(),
	)
}
