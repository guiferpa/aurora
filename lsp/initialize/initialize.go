package initialize

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize

import (
	"encoding/json"

	"github.com/guiferpa/aurora/lsp"
	"github.com/guiferpa/aurora/version"
)

type InitializeRequestParams struct {
	ClientInfo *lsp.ClientInfo `json:"clientInfo"`
}

type InitializeRequest struct {
	lsp.Request
	Params InitializeRequestParams `json:"params"`
}

type InitiazeResult struct {
	ServerCapabilities lsp.ServerCapabilities `json:"capabilities"`
	ServerInfo         lsp.ServerInfo         `json:"serverInfo"`
}

type InitializeResponse struct {
	lsp.Response
	Result InitiazeResult `json:"result"`
}

func ParseRequest(contents []byte) (*InitializeRequest, error) {
	var req InitializeRequest
	if err := json.Unmarshal(contents, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func NewResponse(id int) InitializeResponse {
	return InitializeResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: InitiazeResult{
			ServerCapabilities: lsp.ServerCapabilities{
				TextDocumentSync:   1,
				HoverProvider:      true,
				DefinitionProvider: true,
				CodeActionProvider: true,
				CompletionProvider: map[string]any{},
			},
			ServerInfo: lsp.ServerInfo{
				Name:    "aurorals",
				Version: version.VERSION,
			},
		},
	}
}
