package main

import (
	"log"

	"github.com/guiferpa/aurora/lsp/initialize"
	"github.com/guiferpa/aurora/lsp/state"
)

func InitializeHandler(l *log.Logger, s *state.State, contents []byte) any {
	req, err := initialize.ParseRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}
	client := req.Params.ClientInfo
	l.Printf("Connected to: %s %s", client.Name, client.Version)
	return initialize.NewResponse(req.ID)
}
