package initialize

import (
	"encoding/json"
	"testing"

	"github.com/guiferpa/aurora/version"
)

func TestParseRequest(t *testing.T) {
	req, err := ParseRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"clientInfo":{"name":"nvim","version":"0.11"}}}`))
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.ID != 1 {
		t.Errorf("id = %d, want 1", req.ID)
	}
	if req.Params.ClientInfo == nil {
		t.Fatal("clientInfo was sent and should have been parsed")
	}
	if got, want := req.Params.ClientInfo.Name, "nvim"; got != want {
		t.Errorf("client name = %q, want %q", got, want)
	}
}

// clientInfo is optional in the protocol; a request without it must still parse, and must
// leave a nil that callers can check rather than dereference.
func TestParseRequestWithoutClientInfo(t *testing.T) {
	req, err := ParseRequest([]byte(`{"jsonrpc":"2.0","id":2,"method":"initialize","params":{}}`))
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.Params.ClientInfo != nil {
		t.Errorf("clientInfo = %+v, want nil", req.Params.ClientInfo)
	}
}

func TestParseRequestRejectsGarbage(t *testing.T) {
	if _, err := ParseRequest([]byte("not json")); err == nil {
		t.Error("expected an error")
	}
}

// The response tells the client what the server can do; the legend is what makes coloring
// work, so it has to travel with it.
func TestNewResponse(t *testing.T) {
	res := NewResponse(7)

	if res.RPC != "2.0" {
		t.Errorf("jsonrpc = %q, want 2.0", res.RPC)
	}
	if res.ID == nil || *res.ID != 7 {
		t.Errorf("id = %v, want 7", res.ID)
	}
	if res.Result.ServerInfo.Name != "aurorals" {
		t.Errorf("server name = %q, want aurorals", res.Result.ServerInfo.Name)
	}
	if res.Result.ServerInfo.Version != version.VERSION {
		t.Errorf("server version = %q, want %q", res.Result.ServerInfo.Version, version.VERSION)
	}

	capabilities := res.Result.ServerCapabilities
	if !capabilities.HoverProvider {
		t.Error("hover should be advertised")
	}
	if capabilities.TextDocumentSync != 1 {
		t.Errorf("textDocumentSync = %d, want 1 (full)", capabilities.TextDocumentSync)
	}
	if capabilities.SemanticTokensProvider == nil {
		t.Fatal("semantic tokens should be advertised")
	}
	if !capabilities.SemanticTokensProvider.Full {
		t.Error("the server serves full semantic tokens")
	}
	if len(capabilities.SemanticTokensProvider.Legend.TokenTypes) == 0 {
		t.Error("the legend cannot be empty, or the client has no names for the tokens")
	}
	if !capabilities.DefinitionProvider {
		t.Error("go to definition should be advertised")
	}
	// A capability nobody announces is a feature nobody can use: the client decides what to
	// ask for from this list alone.
	if capabilities.RenameProvider == nil {
		t.Fatal("rename should be advertised")
	}
	if !capabilities.RenameProvider.PrepareProvider {
		t.Error("the server answers prepareRename, which is where a refusal is heard first")
	}
}

// A client reads this as JSON, so the shape on the wire is what matters.
func TestNewResponseMarshals(t *testing.T) {
	bs, err := json.Marshal(NewResponse(1))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(bs, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	result, ok := decoded["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result in %s", bs)
	}
	if _, ok := result["capabilities"].(map[string]any)["semanticTokensProvider"]; !ok {
		t.Errorf("semanticTokensProvider missing from %s", bs)
	}
}

// Snippets are only offered to a client that says it expands them, so the one capability
// the server reads has to survive the parse.
func TestSnippetSupportIsRead(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "a client that expands them",
			body: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{"textDocument":{"completion":{"completionItem":{"snippetSupport":true}}}}}}`,
			want: true,
		},
		{
			name: "a client that says it does not",
			body: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{"textDocument":{"completion":{"completionItem":{"snippetSupport":false}}}}}}`,
			want: false,
		},
		{
			// Everything below capabilities is optional, and a client that says nothing
			// gets plain keywords rather than placeholders in its buffer.
			name: "a client that says nothing at all",
			body: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := ParseRequest([]byte(tc.body))
			if err != nil {
				t.Fatalf("parsing: %v", err)
			}
			if got := req.SnippetSupport(); got != tc.want {
				t.Errorf("SnippetSupport() = %v, want %v", got, tc.want)
			}
		})
	}
}

// The dot is declared as a trigger so the client asks for completion the moment someone
// types it — which is when the fields of a shape are what they want.
func TestDotTriggersCompletion(t *testing.T) {
	result := NewResponse(1).Result.ServerCapabilities.CompletionProvider

	triggers, ok := result["triggerCharacters"].([]string)
	if !ok {
		t.Fatalf("completionProvider has no trigger characters: %v", result)
	}
	if len(triggers) != 1 || triggers[0] != "." {
		t.Errorf("triggers = %v, want the dot", triggers)
	}
}
