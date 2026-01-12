package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"

	"github.com/guiferpa/aurora/builder/evm"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/logger"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/repl"
	"github.com/guiferpa/aurora/version"
)

var (
	player bool
	debug  bool
	raw    bool
	output string
)

var loggers []string

var buildCmd = &cobra.Command{
	Use:   "build [file]",
	Short: "Build binary from source code",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		bs, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		tokens, err := lexer.New(lexer.NewLexerOptions{
			EnableLogging: slices.Contains(loggers, "lexer"),
		}).GetFilledTokens(bs)
		if err != nil {
			return err
		}

		ast, err := parser.New(tokens, parser.NewParserOptions{
			Filename:      filename,
			EnableLogging: slices.Contains(loggers, "parser"),
		}).Parse()
		if err != nil {
			return err
		}

		insts, err := emitter.New(emitter.NewEmitterOptions{
			EnableLogging: slices.Contains(loggers, "emitter"),
		}).Emit(ast)
		if err != nil {
			return err
		}

		fd := os.Stdout
		if strings.Compare(output, "") != 0 {
			fd, err = os.Create(output)
			defer func() {
				err = fd.Close()
			}()
			if err != nil {
				return err
			}
		}
		if _, err := evm.NewBuilder(
			insts,
			evm.NewBuilderOptions{
				EnableLogging: slices.Contains(loggers, "builder"),
			},
		).Build(fd); err != nil {
			return err
		}
		return nil
	},
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Enter in Read-Eval-Print Loop mode",
	Run: func(cmd *cobra.Command, args []string) {
		repl.Start(os.Stdin, os.Stdout, debug, raw, loggers)
	},
}

var runCmd = &cobra.Command{
	Use:   "run [file]",
	Short: "Run program directly from source code",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]
		bs, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		tokens, err := lexer.New(lexer.NewLexerOptions{
			EnableLogging: slices.Contains(loggers, "lexer"),
		}).GetFilledTokens(bs)
		if err != nil {
			return err
		}

		ast, err := parser.New(tokens, parser.NewParserOptions{
			Filename:      filename,
			EnableLogging: slices.Contains(loggers, "parser"),
		}).Parse()
		if err != nil {
			return err
		}

		insts, err := emitter.New(emitter.NewEmitterOptions{
			EnableLogging: slices.Contains(loggers, "emitter"),
		}).Emit(ast)
		if err != nil {
			return err
		}
		ev := evaluator.New(evaluator.NewEvaluatorOptions{
			EnableLogging: slices.Contains(loggers, "evaluator"),
		})
		if player {
			ev.SetPlayer(evaluator.NewPlayer(os.Stdin))
		}
		if _, err := ev.Evaluate(insts); err != nil {
			return err
		}
		logger.AssertError(ev.GetAssertErrors(), filename)
		return nil
	},
}

var deployCmd = &cobra.Command{
	Use:   "deploy [file] [address] [private key]",
	Short: "Deploy program to a blockchain",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]
		bs, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		privateKey, err := crypto.HexToECDSA(args[2])
		if err != nil {
			return err
		}
		from := crypto.PubkeyToAddress(privateKey.PublicKey)

		address := args[1]
		client, err := ethclient.Dial(address)
		if err != nil {
			return err
		}

		nonce, err := client.PendingNonceAt(context.Background(), from)
		if err != nil {
			return err
		}
		gasPrice, err := client.SuggestGasPrice(context.Background())
		if err != nil {
			return err
		}

		tx := types.NewContractCreation(nonce, big.NewInt(0), 3_000_000, gasPrice, bs)

		chainID, err := client.NetworkID(context.Background())
		if err != nil {
			return err
		}
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
		if err != nil {
			return err
		}
		if err := client.SendTransaction(context.Background(), signedTx); err != nil {
			return err
		}

		log.Println("Deploy TX:", signedTx.Hash().Hex())

		contractAddr := crypto.CreateAddress(from, nonce)
		fmt.Println("Contract deployed at:", contractAddr.Hex())
		return nil
	},
}

var callCmd = &cobra.Command{
	Use:   "call [function] [contract address] [address]",
	Short: "Call program on a blockchain",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		fn := args[0]
		selector := crypto.Keccak256([]byte(fn))[:4]
		contract := common.HexToAddress(args[1])

		address := args[2]
		client, err := ethclient.Dial(address)
		if err != nil {
			return err
		}

		message := ethereum.CallMsg{
			To:   &contract,
			Data: selector,
		}

		result, err := client.CallContract(context.Background(), message, nil)
		if err != nil {
			return err
		}

		fmt.Printf("Result: %v\n", result)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show toolbox version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.VERSION)
	},
}

var rootCmd = &cobra.Command{
	Use: "aurora",
}

func main() {
	runCmd.Flags().StringSliceVarP(&loggers, "loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), evaluator)")

	replCmd.Flags().StringSliceVarP(&loggers, "loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), evaluator)")
	replCmd.Flags().BoolVarP(&raw, "raw", "r", false, "enable raw mode for show raw output")

	buildCmd.Flags().StringSliceVarP(&loggers, "loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), builder)")
	buildCmd.Flags().StringVarP(&output, "output", "o", "", "set an output filename")

	rootCmd.AddCommand(versionCmd, runCmd, replCmd, buildCmd, deployCmd, callCmd)

	logger.CommandError(rootCmd.Execute())
}
