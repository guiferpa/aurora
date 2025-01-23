package lsp

type Method string

type URI string

type Request struct {
	RPC    string `json:"jsonrpc"` // value always be 2.0
	ID     int    `json:"id"`
	Method Method `json:"method"`
}

type Response struct {
	RPC string `json:"jsonrpc"` // value always be 2.0
	ID  *int   `json:"id,omitempty"`
}

type Notification struct {
	RPC    string `json:"jsonrpc"` // value always be 2.0
	Method Method `json:"method"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

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
	URI   URI   `json:"uri"`
	Range Range `json:"range"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
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

func LineRange(line, start, end int) Range {
	return Range{
		Start: Position{
			Line:      line,
			Character: start,
		},
		End: Position{
			Line:      line,
			Character: end,
		},
	}
}
