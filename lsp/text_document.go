package lsp

type TextDocumentItem struct {
	/**
	 * The text document's URI.
	 */
	URI string `json:"uri"`

	/**
	* The text document's language identifier.
	 */
	LanguageID string `json:"languageId"`

	/**
	* The version number of this document (it will increase after each
	* change, including undo/redo).
	 */
	Version int `json:"version"`

	/**
	* The content of the opened text document.
	 */
	Text string `json:"text"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentIdentifier
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#versionedTextDocumentIdentifier
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentPositionParams
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type DidOpenTextDocumentNotification struct {
	Notification
	Params DidOpenTextDocumentParams `json:"params"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}
