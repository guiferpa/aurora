package textdoc

import "github.com/guiferpa/aurora/lsp"

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

type CodeActionResponse struct {
	lsp.Response
	Result []CodeAction `json:"result"`
}
