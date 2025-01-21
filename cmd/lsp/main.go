package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/guiferpa/aurora/lsp"
	"github.com/guiferpa/aurora/lsp/analysis"
	"github.com/guiferpa/aurora/lsp/rpc"
	"github.com/guiferpa/aurora/lsp/textdoc"
)

func main() {
	logger := getLogger("/Users/nirdosh/Personal/golsp/markdownlsp.log")
	logger.Println("markdownlsp started...")
	// give scanner something to read from
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	state := analysis.NewState()
	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Printf("failed to decode msg: %s\n", err)
			continue
		}

		handleMessage(logger, writer, state, method, contents)
	}
	logger.Println("markdownlsp stopped")
}

func handleMessage(logger *log.Logger, writer io.Writer, state analysis.State, method string, contents []byte) {
	logger.Printf("received method: %s", method)

	switch method {
	case "initialize":
		var request lsp.InitializeRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Println("initialize err:", err)
			return
		}
		logger.Printf("connected to: %s %s",
			request.Params.ClientInfo.Name,
			request.Params.ClientInfo.Version)

		// now we need to reply initialize reponse to the editor
		msg := lsp.NewInitializeResponse(request.ID)
		writeResponse(writer, msg)
		logger.Println("initialize response sent!")
	case "textDocument/didOpen":
		var request lsp.DidOpenTextDocumentNotification
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Println("textDocument/didOpen err:", err)
			return
		}
		logger.Printf("opened: %s\n", request.Params.TextDocument.URI)
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
			logger.Println("textDocument/didChange err:", err)
			return
		}

		logger.Printf("changed: %s\n", request.Params.TextDocument.URI)
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
			logger.Println("textDocument/codeAction err:", err)
			return
		}

		response := state.TextDocumentCodeAction(request.ID, request.Params.TextDocument.URI)
		writeResponse(writer, response)

	case "textDocument/completion":
		var request textdoc.CompletionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Println("textDocument/completion err:", err)
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

func getLogger(filename string) *log.Logger {
	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic("faild to initialize logfile" + err.Error())
	}

	return log.New(logfile, "[markdownlsp]", log.Ldate|log.Ltime|log.Lshortfile)
}
