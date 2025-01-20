package lsp

type Request struct {
	RPC    string `json:"jsonrpc"` // value always be 2.0
	ID     int    `json:"id"`
	Method string `json:"method"`
}

type Response struct {
	RPC string `json:"jsonrpc"` // value always be 2.0
	ID  *int   `json:"id,omitempty"`
}

type Notification struct {
	RPC    string `json:"jsonrpc"` // value always be 2.0
	Method string `json:"method"`
}
