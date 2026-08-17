package textdoc

import (
	"encoding/json"

	"github.com/guiferpa/aurora/hosting/lsp"
)

type DidCloseParams struct {
	TextDocument Identifier `json:"textDocument"`
}

type DidCloseNotification struct {
	lsp.Notification
	Params DidCloseParams `json:"params"`
}

func ParseDidCloseNotification(contents []byte) (*DidCloseNotification, error) {
	var noti DidCloseNotification
	if err := json.Unmarshal(contents, &noti); err != nil {
		return nil, err
	}
	return &noti, nil
}
