package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/guiferpa/aurora/lsp"
	"github.com/guiferpa/aurora/lsp/analysis"
	"github.com/guiferpa/aurora/lsp/rpc"
	"github.com/guiferpa/aurora/lsp/textdoc"
)

func main() {
	file, err := os.Create("lsp.log")
	fmt.Println(err)
	logger := log.New(file, "[aurora] ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Println("Server started")

	// give scanner something to read from
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	state := analysis.NewState()
	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		logger.Println("======>", string(msg))
		if err != nil {
			logger.Printf("Failed to decode message: %s\n", err)
			continue
		}

		handleMessage(logger, writer, state, method, contents)
	}
	logger.Println("Server stopped")
}

func handleMessage(logger *log.Logger, writer io.Writer, state analysis.State, method string, contents []byte) {
	logger.Printf("Received method: %s", method)

	switch method {
	case "initialize":
		var request lsp.InitializeRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Println("Method initialize error:", err)
			return
		}
		logger.Printf("Connected to: %s %s",
			request.Params.ClientInfo.Name,
			request.Params.ClientInfo.Version)

		// now we need to reply initialize reponse to the editor
		msg := lsp.NewInitializeResponse(request.ID)
		if err := json.NewEncoder(logger.Writer()).Encode(msg); err != nil {
			logger.Println(err)
		}
		writeResponse(writer, msg)
		logger.Println("Initialize response sent!")

		/*

			case "textDocument/didOpen":
				var request lsp.DidOpenTextDocumentNotification
				if err := json.Unmarshal(contents, &request); err != nil {
					logger.Println("Method textDocument/didOpen error:", err)
					return
				}
				logger.Printf("Opened: %s\n", request.Params.TextDocument.URI)
				diagnostics := state.OpenDocument(request.Params.TextDocument.URI, request.Params.TextDocument.Text)
				writeResponse(writer, textdoc.DiagnosticsNotification{
					Notification: lsp.Notification{
						RPC:    "2.0",
						Method: "textDocument/publishDiagnostics",
					},
					Params: textdoc.DiagnosticsParams{
						URI:         request.Params.TextDocument.URI,
						Diagnostics: diagnostics,
					},
				})

			case "textDocument/didChange":
				var request textdoc.DidChangeNotification
				if err := json.Unmarshal(contents, &request); err != nil {
					logger.Println("Method textDocument/didChange error:", err)
					return
				}

				logger.Printf("Changed: %s\n", request.Params.TextDocument.URI)
				for _, change := range request.Params.ContentChanges {

					diagnostics := state.UpdateDocument(request.Params.TextDocument.URI, change.Text)
					writeResponse(writer, textdoc.DiagnosticsNotification{
						Notification: lsp.Notification{
							RPC:    "2.0",
							Method: "textDocument/publishDiagnostics",
						},
						Params: textdoc.DiagnosticsParams{
							URI:         request.Params.TextDocument.URI,
							Diagnostics: diagnostics,
						},
					})
				}
			case "textDocument/hover":
				var request lsp.HoverRequest
				if err := json.Unmarshal(contents, &request); err != nil {
					logger.Println("textDocument/hover err:", err)
					return
				}

				// create a response which will be displayed by the editor while hovering
				response := state.Hover(request.ID, request.Params.TextDocument.URI, request.Params.Position)
				writeResponse(writer, response)
			case "textDocument/definition":
				var request textdoc.DefinitionRequest
				if err := json.Unmarshal(contents, &request); err != nil {
					logger.Println("textDocument/definition err:", err)
					return
				}

				response := state.Definition(request.ID, request.Params.TextDocument.URI, request.Params.Position)
				writeResponse(writer, response)
			case "textDocument/codeAction": // leader + ca
				var request textdoc.CodeActionRequest
				if err := json.Unmarshal(contents, &request); err != nil {
					logger.Println("Method textDocument/codeAction err:", err)
					return
				}

				response := state.TextDocumentCodeAction(request.ID, request.Params.TextDocument.URI)
				writeResponse(writer, response)

		*/
	case "textDocument/completion":
		var request textdoc.CompletionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Println("Method textDocument/completion error:", err)
			return
		}

		// in reality, we would be passing the position as well to perform other string manipulations
		response := state.TextDocumentCompletion(request.ID, request.Params.TextDocument.URI)
		writeResponse(writer, response)
	}
}

// write message to given writer
// writer can be anything. e.g. stdout or http response
func writeResponse(writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)
	writer.Write([]byte(reply))
}
