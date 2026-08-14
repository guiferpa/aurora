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
	// clientInfo is optional in the protocol, so it arrives nil from any client that does
	// not send it. Dereferencing it took the whole server down on the very first message.
	if client := req.Params.ClientInfo; client != nil {
		l.Printf("Connected to: %s %s", client.Name, client.Version)
	}
	return initialize.NewResponse(req.ID)
}
