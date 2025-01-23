package textdoc

import (
	"encoding/json"

	"github.com/guiferpa/aurora/lsp"
)

type DefinitionRequest struct {
	lsp.Request
	Params DefinitionParams `json:"params"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#definitionParams
type DefinitionParams struct {
	PositionParams
}

type DefinitionResponse struct {
	lsp.Response
	Result lsp.Location `json:"result"`
}

func ParseDefinitionRequest(contents []byte) (*DefinitionRequest, error) {
	var req DefinitionRequest
	if err := json.Unmarshal(contents, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func NewDefinitionResponse(id int, uri lsp.URI, position lsp.Position) DefinitionResponse {
	return DefinitionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: lsp.Location{
			URI: uri,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      position.Line - 1,
					Character: 0,
				},
				End: lsp.Position{
					Line:      position.Line - 1,
					Character: 0,
				},
			},
		},
	}
}
