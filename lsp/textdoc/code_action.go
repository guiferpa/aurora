package textdoc

import (
	"encoding/json"

	"github.com/guiferpa/aurora/lsp"
)

type CodeActionRequest struct {
	lsp.Request
	Params CodeActionParams `json:"params"`
}

type CodeActionParams struct {
	TextDocument Identifier        `json:"textDocument"`
	Range        lsp.Range         `json:"range"`
	Context      CodeActionContext `json:"context"`
}

type CodeActionContext struct {
	// add fields as needed
}

type CodeAction struct {
	Title   string         `json:"title"`
	Edit    *WorkspaceEdit `json:"edit,omitempty"`
	Command *Command       `json:"command,omitempty"`
}

type CodeActions []CodeAction

type CodeActionResult CodeActions

type CodeActionResponse struct {
	lsp.Response
	Result CodeActionResult `json:"result"`
}

func ParseCodeActionRequest(contents []byte) (*CodeActionRequest, error) {
	var req CodeActionRequest
	if err := json.Unmarshal(contents, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func NewCodeActionResponse(id int, cas CodeActions) CodeActionResponse {
	return CodeActionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: CodeActionResult(cas),
	}
}
