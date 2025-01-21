package textdoc

import "github.com/guiferpa/aurora/lsp"

type DidOpenParams struct {
	TextDocument Item `json:"textDocument"`
}

type DidOpenNotification struct {
	lsp.Notification
	Params DidOpenParams `json:"params"`
}
