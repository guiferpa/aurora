package textdoc

import (
	"encoding/json"

	"github.com/guiferpa/aurora/lsp"
)

type DidOpenParams struct {
	TextDocument Item `json:"textDocument"`
}

type DidOpenNotification struct {
	lsp.Notification
	Params DidOpenParams `json:"params"`
}

func ParseDidOpenNotification(contents []byte) (*DidOpenNotification, error) {
	var noti DidOpenNotification
	if err := json.Unmarshal(contents, &noti); err != nil {
		return nil, err
	}
	return &noti, nil
}
