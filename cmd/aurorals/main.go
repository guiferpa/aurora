package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/version"
)

func main() {
	logPath := flag.String("log", "", "write server logs to this file (default: stderr)")
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.VERSION)
		return
	}

	// stdin and stdout carry the protocol, so logs go to stderr, which clients capture.
	// A file is opt-in: writing one unconditionally would litter the editor's working
	// directory and kill the server where that directory is read-only.
	var out io.Writer = os.Stderr
	if *logPath != "" {
		file, err := os.Create(*logPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "aurorals: cannot open log file: %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = file.Close() }()
		out = file
	}

	logger := log.New(out, "[aurorals] ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Printf("Server started (version %s)", version.VERSION)

	lsp.Listen(logger, os.Stdin, os.Stdout, handlers())

	logger.Println("Server stopped")
}

// handlers maps the methods the server implements. shutdown and exit are answered by the
// listener itself.
func handlers() map[lsp.Method]lsp.MethodHandler {
	return map[lsp.Method]lsp.MethodHandler{
		"initialize":                       InitializeHandler,
		"textDocument/didOpen":             TextdocDidOpenHandler,
		"textDocument/didChange":           TextdocDidChangeHandler,
		"textDocument/didClose":            TextdocDidCloseHandler,
		"textDocument/completion":          TextdocCompletionHandler,
		"textDocument/hover":               TextdocHoverHandler,
		"textDocument/semanticTokens/full": TextdocSemanticTokensHandler,
	}
}
