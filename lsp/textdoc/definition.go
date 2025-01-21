package textdoc

import "github.com/guiferpa/aurora/lsp"

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
