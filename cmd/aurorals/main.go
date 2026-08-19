package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/hosting/lsp/state"
	"github.com/guiferpa/aurora/hosting/lsp/textdoc"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
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

	// The phases are built here, once, and the session that holds them answers every request.
	// A document carries the width of the project it belongs to, which is why one session
	// serves whatever the editor opens.
	//
	// What a client has open is made here too, because the module resolution needs it: a
	// file it has to read may be one the person is editing, and the version that counts is
	// theirs rather than the disk's.
	documents := state.New()
	sv := server{textdoc: textdoc.NewSession(textdoc.NewSessionOptions{
		Lexer:   lexer.New(),
		Parser:  parser.New(),
		Resolve: resolveModules(documents),
	})}

	lsp.Listen(logger, os.Stdin, os.Stdout, documents, sv.handlers())

	logger.Println("Server stopped")
}

// A server is what answers the editor: the session that reads a document, and the handlers
// that are methods on it. It exists so the phases are built once, in main, rather than by
// whoever happens to be answering a request.
type server struct {
	textdoc *textdoc.Session
}

// handlers maps the methods the server implements. shutdown and exit are answered by the
// listener itself.
func (sv server) handlers() map[lsp.Method]lsp.MethodHandler {
	return map[lsp.Method]lsp.MethodHandler{
		"initialize":                       InitializeHandler,
		"textDocument/didOpen":             sv.didOpen,
		"textDocument/didChange":           sv.didChange,
		"textDocument/didClose":            sv.didClose,
		"textDocument/completion":          sv.completion,
		"textDocument/hover":               sv.hover,
		"textDocument/semanticTokens/full": sv.semanticTokens,
	}
}
