package lsp

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize

type InitializeRequestParams struct {
	ClientInfo *ClientInfo `json:"clientInfo"`
}

type InitializeRequest struct {
	Request
	Params InitializeRequestParams `json:"params"`
}

type InitiazeResult struct {
	ServerCapabilities ServerCapabilities
	ServerInfo         ServerInfo
}

type InitializeResponse struct {
	Response
	Result InitiazeResult `json:"result"`
}

func NewInitializeResponse(id int) InitializeResponse {
	return InitializeResponse{
		Response: Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: InitiazeResult{
			ServerCapabilities: ServerCapabilities{
				TextDocumentSync:   1,
				HoverProvider:      true,
				DefinitionProvider: true,
				CodeActionProvider: true,
				CompletionProvider: map[string]any{},
			},
			ServerInfo: ServerInfo{
				Name:    "",
				Version: "",
			},
		},
	}
}
