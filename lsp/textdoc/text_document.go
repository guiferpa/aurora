package textdoc

import "github.com/guiferpa/aurora/lsp"

type Item struct {
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
type Identifier struct {
	URI string `json:"uri"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#versionedTextDocumentIdentifier
type VersionedIdentifier struct {
	Identifier
	Version int `json:"version"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentPositionParams
type PositionParams struct {
	TextDocument Identifier   `json:"textDocument"`
	Position     lsp.Position `json:"position"`
}

type ContentChangeEvent struct {
	// The new text for the provided range.
	Text string `json:"text"`
}

type Command struct {
	Title     string `json:"title"`
	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}

// contains changes to be made in a bunch of files
// replaces old text from given range with new text for given files
// one file can have multiple text edits
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
}

type TextEdit struct {
	Range   lsp.Range `json:"range"`
	NewText string    `json:"newText"`
}
