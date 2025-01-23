package textdoc

import (
	"encoding/json"

	"github.com/guiferpa/aurora/lsp"
)

type CompletionParams struct {
	PositionParams
}

type CompletionRequest struct {
	lsp.Request
	Params CompletionParams `json:"params"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionItem
type CompletionItem struct {
	Label         string `json:"label"`
	Detail        string `json:"detail"`
	Documentation string `json:"documentation"`
}

type CompletionResult struct {
	Items []CompletionItem `json:"items"`
}

type CompletionResponse struct {
	lsp.Response
	Result CompletionResult `json:"result"`
}

func ParseCompletionRequest(contents []byte) (*CompletionRequest, error) {
	var req CompletionRequest
	if err := json.Unmarshal(contents, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func NewCompletionResponse(id int, items []CompletionItem) CompletionResponse {
	return CompletionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: CompletionResult{
			Items: items,
		},
	}
}
