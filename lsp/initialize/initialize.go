package initialize

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize

import (
	"encoding/json"

	"github.com/guiferpa/aurora/lsp"
	"github.com/guiferpa/aurora/lsp/textdoc"
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
				// 1 = full sync: the client resends the whole document on each change.
				TextDocumentSync:   1,
				HoverProvider:      true,
				CompletionProvider: map[string]any{},
				// The legend is what makes coloring work in any client: the server
				// reports token types the lexer already knows, instead of every editor
				// keeping a grammar of its own in sync.
				SemanticTokensProvider: &lsp.SemanticTokensOptions{
					Legend: lsp.SemanticTokensLegend{
						TokenTypes:     textdoc.SemanticTokenTypes,
						TokenModifiers: textdoc.SemanticTokenModifiers,
					},
					Full: true,
				},
			},
			ServerInfo: lsp.ServerInfo{
				Name:    "aurorals",
				Version: version.VERSION,
			},
		},
	}
}
