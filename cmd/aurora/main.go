package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
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
	"github.com/guiferpa/aurora/manifest"
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
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := manifest.FindProjectRoot()
		if err != nil {
			return err
		}
		m, err := manifest.Load(root)
		if err != nil {
			return err
		}
		prof, err := m.Profile("main")
		if err != nil {
			return err
		}

		filename := manifest.AbsPath(root, prof.Entrypoint)
		if len(args) > 0 {
			filename = args[0]
		}

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

		outPath := output
		if outPath == "" {
			outPath = manifest.AbsPath(root, prof.Target)
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		fd, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer func() {
			err = fd.Close()
		}()
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
		repl.Start(os.Stdin, debug, raw, loggers)
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
			EchoWriter:    ToMainWriter(),
			PrintWriter:   ToMainWriter(),
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
	Use:   "deploy",
	Short: "Deploy program to a blockchain (reads target, rpc_url, private_key_path from manifest)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := manifest.FindProjectRoot()
		if err != nil {
			return err
		}
		m, err := manifest.Load(root)
		if err != nil {
			return err
		}
		prof, err := m.Profile("main")
		if err != nil {
			return err
		}
		if prof.RPCURL == "" {
			return fmt.Errorf("profile main: rpc_url is required for deploy")
		}
		if prof.PrivateKeyPath == "" {
			return fmt.Errorf("profile main: private_key_path is required for deploy")
		}

		bytecodePath := manifest.AbsPath(root, prof.Target)
		bs, err := os.ReadFile(bytecodePath)
		if err != nil {
			return fmt.Errorf("read bytecode from %s: %w", bytecodePath, err)
		}

		keyPath := manifest.AbsPath(root, prof.PrivateKeyPath)
		keyHex, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("read private key from %s: %w", keyPath, err)
		}
		privateKey, err := crypto.HexToECDSA(strings.TrimSpace(string(keyHex)))
		if err != nil {
			return err
		}
		from := crypto.PubkeyToAddress(privateKey.PublicKey)

		client, err := ethclient.Dial(prof.RPCURL)
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
	Use:   "call <function> [profile]",
	Short: "Call program on a blockchain (reads contract_address, rpc_url from manifest)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fn := args[0]
		profileName := "main"
		if len(args) > 1 {
			profileName = args[1]
		}

		root, err := manifest.FindProjectRoot()
		if err != nil {
			return err
		}
		m, err := manifest.Load(root)
		if err != nil {
			return err
		}
		prof, err := m.Profile(profileName)
		if err != nil {
			return err
		}
		if prof.RPCURL == "" {
			return fmt.Errorf("profile %s: rpc_url is required for call", profileName)
		}
		if prof.ContractAddress == "" {
			return fmt.Errorf("profile %s: contract_address is required for call", profileName)
		}

		selector := crypto.Keccak256([]byte(fn))[:4]
		contract := common.HexToAddress(prof.ContractAddress)

		client, err := ethclient.Dial(prof.RPCURL)
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

const initManifestTemplate = `# Aurora project manifest.
# See https://github.com/guiferpa/aurora for more information.

[project]
# Project identifier (inherited from the root folder name where 'aurora init' was run).
name = %q
# Project version (semantic version recommended).
version = "0.1.0"

[profiles.main]
# Default profile. Commands like 'aurora build' or 'aurora run' use these paths when no file is given.
# Path to the main source file (entrypoint). Used by build, run, and deploy when no file argument is passed.
entrypoint = "src/main.ar"
# Path where the compiled binary is written. Name matches the entrypoint filename (without extension). Used by 'aurora build' when no -o output is passed.
target = "dist/main"
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create an aurora.toml manifest in the current directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(manifest.Filename); err == nil {
			return fmt.Errorf("%s already exists", manifest.Filename)
		}
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		projectName := filepath.Base(dir)
		content := fmt.Sprintf(initManifestTemplate, projectName)
		return os.WriteFile(manifest.Filename, []byte(content), 0o644)
	},
}

var rootCmd = &cobra.Command{
	Use: "aurora",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		switch args[0] {
		case "init", "version", "help", "repl":
			return nil
		}
		return requireManifest()
	},
}

// requireManifest ensures aurora.toml exists in the current directory or any parent.
// Must be called before commands that need a project (all except init and version).
func requireManifest() error {
	_, err := manifest.FindProjectRoot()
	return err
}

func main() {
	runCmd.Flags().StringSliceVarP(&loggers, "loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), evaluator)")

	replCmd.Flags().StringSliceVarP(&loggers, "loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), evaluator)")
	replCmd.Flags().BoolVarP(&raw, "raw", "r", false, "enable raw mode for show raw output")

	buildCmd.Flags().StringSliceVarP(&loggers, "loggers", "l", []string{}, "enable loggers for show deep dive logs from all phases (valid: lexer, parser, emitter (not implemented yet), builder)")
	buildCmd.Flags().StringVarP(&output, "output", "o", "", "output path for compiled binary (default: target from manifest)")

	rootCmd.AddCommand(versionCmd, runCmd, replCmd, buildCmd, deployCmd, callCmd, initCmd)

	logger.CommandError(rootCmd.Execute())
}
