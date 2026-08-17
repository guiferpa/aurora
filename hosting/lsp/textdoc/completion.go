package textdoc

import (
	"encoding/json"

	"github.com/guiferpa/aurora/hosting/lsp"
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

// InsertTextFormat says how InsertText is read by the client. (Snippet is taken here by the
// item kind of the same name, so these carry the suffix.)
const (
	PlainTextFormat = 1
	SnippetFormat   = 2
)

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionItem
type CompletionItem struct {
	Label         string             `json:"label"`
	Detail        string             `json:"detail"`
	Documentation string             `json:"documentation"`
	Kind          CompletionItemKind `json:"kind"`
	// InsertText is what lands in the buffer, when it differs from the label — a snippet
	// with the places to fill in. Left empty, the client inserts the label.
	InsertText string `json:"insertText,omitempty"`
	// InsertTextFormat is Snippet only for a client that said it expands them: the
	// placeholders are literal text to anyone else.
	InsertTextFormat int `json:"insertTextFormat,omitempty"`
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
