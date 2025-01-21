package lsp

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#serverCapabilities
type ServerCapabilities struct {
	TextDocumentSync   int            `json:"textDocumentSync"`
	HoverProvider      bool           `json:"hoverProvider"`
	DefinitionProvider bool           `json:"definitionProvider"`
	CodeActionProvider bool           `json:"codeActionProvider"`
	CompletionProvider map[string]any `json:"completionProvider"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#position
type Position struct {
	// index of line cursor is in. index of first line in the file is 0.
	Line int `json:"line"`
	// index of character in the line where the cursor is. starts from zero
	Character int `json:"character"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#location
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type HoverRequest struct {
	Request
	Params HoverParams `json:"params"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#hoverParams
type HoverParams struct {
	TextDocumentPositionParams
}

type HoverResponse struct {
	Response
	Result HoverResult `json:"result"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#hover
type HoverResult struct {
	Contents string `json:"contents"`
}

type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// contains changes to be made in a bunch of files
// replaces old text from given range with new text for given files
// one file can have multiple text edits
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}
