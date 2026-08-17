package initialize

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize

import (
	"encoding/json"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/hosting/lsp/textdoc"
	"github.com/guiferpa/aurora/version"
)

type InitializeRequestParams struct {
	ClientInfo   *lsp.ClientInfo    `json:"clientInfo"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

// ClientCapabilities carries the one thing the server changes its answers for: whether the
// client expands snippets. Everything else it reports is ignored.
type ClientCapabilities struct {
	TextDocument struct {
		Completion struct {
			CompletionItem struct {
				SnippetSupport bool `json:"snippetSupport"`
			} `json:"completionItem"`
		} `json:"completion"`
	} `json:"textDocument"`
}

// SnippetSupport says whether the client expands ${1:placeholders}.
func (r InitializeRequest) SnippetSupport() bool {
	return r.Params.Capabilities.TextDocument.Completion.CompletionItem.SnippetSupport
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
				TextDocumentSync: 1,
				HoverProvider:    true,
				// The dot is declared as a trigger so a client asks for completion the
				// moment someone types it — which is when the fields of a struct are
				// what they want, and the only moment the document does not parse.
				CompletionProvider: map[string]any{
					"triggerCharacters": []string{"."},
				},
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
