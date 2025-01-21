package textdoc

import "github.com/guiferpa/aurora/lsp"

// diagnostic is a push notification form the language server. so there is no request/response cycle

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#diagnostic
type Diagnostic struct {
	// The range at which the message applies.
	Range    lsp.Range `json:"range"`
	Severity int       `json:"severity"`
	Source   string    `json:"source"`
	// displayed to the user
	Message string `json:"message"`
}

type Diagnocstics []Diagnostic

type DiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics Diagnocstics `json:"diagnostics"`
}

type DiagnosticsNotification struct {
	lsp.Notification
	Params DiagnosticsParams `json:"params"`
}
