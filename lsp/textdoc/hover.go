package textdoc

import (
	"encoding/json"

	"github.com/guiferpa/aurora/lsp"
)

type HoverRequest struct {
	lsp.Request
	Params HoverParams `json:"params"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#hoverParams
type HoverParams struct {
	PositionParams
}

type HoverResponse struct {
	lsp.Response
	Result HoverResult `json:"result"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#hover
type HoverResult struct {
	Contents string `json:"contents"`
}

func ParseHoverRequest(contents []byte) (*HoverRequest, error) {
	var req HoverRequest
	if err := json.Unmarshal(contents, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func NewHoverResponse(id int, contents string) HoverResponse {
	return HoverResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: HoverResult{
			Contents: contents,
		},
	}
}
