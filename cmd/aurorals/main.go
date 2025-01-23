package main

import (
	"log"
	"os"

	"github.com/guiferpa/aurora/lsp"
)

func main() {
	file, err := os.Create("lsp.log")
	if err != nil {
		log.Panic(err)
	}
	logger := log.New(file, "[aurorals] ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Println("Server started")

	lsp.Listen(logger, os.Stdin, os.Stdout, map[lsp.Method]lsp.MethodHandler{
		"initialize":              InitializeHandler,
		"textDocument/completion": TextdocCompletionHandler,
		"textDocument/didOpen":    TextdocDidOpenHandler,
		"textDocument/didChange":  TextdocDidChangeHandler,
		"textDocument/hover":      TextdocHoverHandler,
		"textDocument/definition": TextdocDefinitionHandler,
		"textDocument/codeAction": TextdocCodeActionHandler,
	})

	logger.Println("Server stopped")
}
