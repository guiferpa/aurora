package lsp

type Method string

// Lifecycle methods the listener answers on its own.
const (
	MethodShutdown Method = "shutdown"
	MethodExit     Method = "exit"
)

// JSON-RPC error codes used by the server.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#responseMessage
const CodeMethodNotFound = -32601

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

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse answers a request the server cannot handle.
type ErrorResponse struct {
	Response
	Error ResponseError `json:"error"`
}

// NullResponse answers a request whose result is null (shutdown).
type NullResponse struct {
	Response
	Result any `json:"result"`
}

func NewMethodNotFoundResponse(id int, method string) ErrorResponse {
	return ErrorResponse{
		Response: Response{RPC: "2.0", ID: &id},
		Error:    ResponseError{Code: CodeMethodNotFound, Message: "method not supported: " + method},
	}
}

func NewNullResponse(id *int) NullResponse {
	return NullResponse{Response: Response{RPC: "2.0", ID: id}, Result: nil}
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// SemanticTokensLegend tells the client how to read the token type and modifier indexes
// the server sends. https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#semanticTokensLegend
type SemanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

type SemanticTokensOptions struct {
	Legend SemanticTokensLegend `json:"legend"`
	Full   bool                 `json:"full"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#serverCapabilities
type ServerCapabilities struct {
	TextDocumentSync       int                    `json:"textDocumentSync"`
	HoverProvider          bool                   `json:"hoverProvider"`
	DefinitionProvider     bool                   `json:"definitionProvider"`
	CompletionProvider     map[string]any         `json:"completionProvider"`
	SemanticTokensProvider *SemanticTokensOptions `json:"semanticTokensProvider,omitempty"`
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
