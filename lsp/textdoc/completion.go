package textdoc

import (
	"encoding/json"

	"github.com/guiferpa/aurora/lsp"
)

type CompletionItemKind uint8

const (
	Text CompletionItemKind = iota + 1
	Method
	Function
	Constructor
	Field
	Variable
	Class
	Interface
	Module
	Property
	Unit
	Value
	Enum
	Keyword
	Snippet
	Color
	File
	Reference
	Folder
	EnumMember
	Constant
	Struct
	Event
	Operator
	TypeParameter
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
	Label         string             `json:"label"`
	Detail        string             `json:"detail"`
	Documentation string             `json:"documentation"`
	Kind          CompletionItemKind `json:"kind"`
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
