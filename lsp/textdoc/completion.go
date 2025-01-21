package textdoc

import "github.com/guiferpa/aurora/lsp"

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

type CompletionResponse struct {
	lsp.Response
	Result []CompletionItem `json:"result"`
}
