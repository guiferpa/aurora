package textdoc

import "github.com/guiferpa/aurora/lsp"

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#didChangeTextDocumentParams
type DidChangeParams struct {
	TextDocument   VersionedIdentifier  `json:"textDocument"`
	ContentChanges []ContentChangeEvent `json:"contentChanges"`
}

type DidChangeNotification struct {
	lsp.Notification
	Params DidChangeParams `json:"params"`
}
