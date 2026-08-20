package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/hosting/lsp/state"
	"github.com/guiferpa/aurora/hosting/lsp/textdoc"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

// A scripted session over the real handler map: what a client actually exchanges with the
// server, from initialize to exit.

func frame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

// runSession feeds the messages to the server and returns every decoded reply.
func runSession(t *testing.T, messages ...string) []map[string]any {
	t.Helper()

	in := strings.NewReader(strings.Join(messages, ""))
	out := bytes.NewBuffer(nil)
	sv := server{textdoc: textdoc.NewSession(textdoc.NewSessionOptions{
		Lexer:  lexer.New(),
		Parser: parser.New(),
	})}
	lsp.Listen(log.New(io.Discard, "", 0), in, out, state.New(), sv.handlers())

	replies := make([]map[string]any, 0)
	rest := out.Bytes()
	for len(rest) > 0 {
		_, body, found := bytes.Cut(rest, []byte("\r\n\r\n"))
		if !found {
			t.Fatalf("malformed reply stream: %q", rest)
		}
		decoder := json.NewDecoder(bytes.NewReader(body))
		var reply map[string]any
		if err := decoder.Decode(&reply); err != nil {
			t.Fatalf("decoding reply: %v", err)
		}
		replies = append(replies, reply)
		rest = body[decoder.InputOffset():]
	}
	return replies
}

func didOpen(uri, text string) string {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri, "languageId": "aurora", "version": 1, "text": text},
		},
	})
	return frame(string(body))
}

func request(id int, method string, params map[string]any) string {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	})
	return frame(string(body))
}

const exitMessage = "Content-Length: 33\r\n\r\n{\"jsonrpc\":\"2.0\",\"method\":\"exit\"}"

func TestSessionInitializeAdvertisesSemanticTokens(t *testing.T) {
	replies := runSession(t,
		request(1, "initialize", map[string]any{"clientInfo": map[string]any{"name": "test", "version": "1"}}),
		exitMessage,
	)

	if len(replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(replies))
	}
	result := replies[0]["result"].(map[string]any)
	capabilities := result["capabilities"].(map[string]any)

	provider, ok := capabilities["semanticTokensProvider"].(map[string]any)
	if !ok {
		t.Fatalf("semanticTokensProvider missing from capabilities: %v", capabilities)
	}
	legend := provider["legend"].(map[string]any)
	types := legend["tokenTypes"].([]any)
	if len(types) == 0 || types[0] != "keyword" {
		t.Errorf("unexpected legend: %v", types)
	}
	if provider["full"] != true {
		t.Error("the server must advertise full semantic tokens")
	}
	if capabilities["hoverProvider"] != true {
		t.Error("hover should be advertised")
	}
}

// clientInfo is optional in the protocol. A client that omits it used to take the server
// down with a nil dereference on the first message it ever received.
func TestSessionInitializeWithoutClientInfo(t *testing.T) {
	replies := runSession(t,
		request(1, "initialize", map[string]any{}),
		exitMessage,
	)

	if len(replies) != 1 {
		t.Fatalf("expected the server to answer, got %d replies", len(replies))
	}
	if _, ok := replies[0]["result"].(map[string]any)["capabilities"]; !ok {
		t.Errorf("expected capabilities in the reply: %v", replies[0])
	}
}

func TestSessionPublishesDiagnosticsOnOpenAndClearsThemOnFix(t *testing.T) {
	uri := "file:///tmp/main.ar"
	fixed, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didChange",
		"params": map[string]any{
			"textDocument":   map[string]any{"uri": uri, "version": 2},
			"contentChanges": []map[string]any{{"text": "ident a = 1;\n"}},
		},
	})

	replies := runSession(t,
		didOpen(uri, "ident a = ;\n"),
		frame(string(fixed)),
		exitMessage,
	)

	if len(replies) != 2 {
		t.Fatalf("expected a diagnostics notification per change, got %d", len(replies))
	}

	first := replies[0]["params"].(map[string]any)["diagnostics"].([]any)
	if len(first) != 1 {
		t.Fatalf("expected the broken document to report one problem, got %v", first)
	}
	if replies[0]["method"] != "textDocument/publishDiagnostics" {
		t.Errorf("method = %v", replies[0]["method"])
	}

	second := replies[1]["params"].(map[string]any)["diagnostics"].([]any)
	if len(second) != 0 {
		t.Errorf("fixing the document must clear the diagnostics, got %v", second)
	}
}

func TestSessionSemanticTokensForOpenDocument(t *testing.T) {
	uri := "file:///tmp/main.ar"
	replies := runSession(t,
		didOpen(uri, "ident a = 1;\n"),
		request(2, "textDocument/semanticTokens/full", map[string]any{
			"textDocument": map[string]any{"uri": uri},
		}),
		exitMessage,
	)

	if len(replies) != 2 {
		t.Fatalf("expected diagnostics plus the token reply, got %d", len(replies))
	}
	data := replies[1]["result"].(map[string]any)["data"].([]any)
	if len(data) == 0 || len(data)%5 != 0 {
		t.Fatalf("token data must be a non-empty multiple of 5, got %v", data)
	}
	// First token: the "ident" keyword at line 0, character 0, five units long.
	if data[0] != float64(0) || data[1] != float64(0) || data[2] != float64(5) {
		t.Errorf("first token = %v, want line 0 char 0 length 5", data[:5])
	}
}

func TestSessionHoverUsesTheKeywordDescription(t *testing.T) {
	uri := "file:///tmp/main.ar"
	replies := runSession(t,
		didOpen(uri, "ident a = 1;\n"),
		request(3, "textDocument/hover", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 2},
		}),
		exitMessage,
	)

	if len(replies) != 2 {
		t.Fatalf("expected diagnostics plus the hover reply, got %d", len(replies))
	}
	contents := replies[1]["result"].(map[string]any)["contents"]
	if !strings.Contains(fmt.Sprint(contents), "identifier") {
		t.Errorf("hover contents = %v", contents)
	}
}

func TestSessionCompletionOffersKeywords(t *testing.T) {
	uri := "file:///tmp/main.ar"
	replies := runSession(t,
		didOpen(uri, "ident total = 1;\n"),
		request(4, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 1, "character": 0},
		}),
		exitMessage,
	)

	items := replies[1]["result"].(map[string]any)["items"].([]any)
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, item.(map[string]any)["label"].(string))
	}
	joined := strings.Join(labels, ",")
	for _, want := range []string{"ident", "defer", "total"} {
		if !strings.Contains(joined, want) {
			t.Errorf("completion is missing %q, got %v", want, labels)
		}
	}
}

// The whole round trip of a jump: a document opened, a position asked about, and a location
// coming back — with the URI the client knows the file by, which is this side's half of the
// answer.
func TestSessionDefinitionAnswersWithALocation(t *testing.T) {
	uri := "file:///tmp/main.ar"
	replies := runSession(t,
		didOpen(uri, "ident total = 1;\nprintd total;\n"),
		request(5, "textDocument/definition", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 1, "character": 8},
		}),
		exitMessage,
	)

	if len(replies) != 2 {
		t.Fatalf("expected diagnostics plus the definition reply, got %d", len(replies))
	}
	result := replies[1]["result"].(map[string]any)
	if result["uri"] != uri {
		t.Errorf("points at %v, want %s", result["uri"], uri)
	}
	start := result["range"].(map[string]any)["start"].(map[string]any)
	if start["line"] != float64(0) || start["character"] != float64(6) {
		t.Errorf("points at %v, want line 0 character 6", start)
	}
}

// A position with no name under it is answered with null rather than with nothing. A request
// carries an id and the client waits on it: silence is a client that waits forever.
func TestSessionDefinitionOfNothingIsNull(t *testing.T) {
	uri := "file:///tmp/main.ar"
	replies := runSession(t,
		didOpen(uri, "printd 42;\n"),
		request(6, "textDocument/definition", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 8},
		}),
		exitMessage,
	)

	if len(replies) != 2 {
		t.Fatalf("expected diagnostics plus the null reply, got %d replies: %v", len(replies), replies)
	}
	result, answered := replies[1]["result"]
	if !answered || result != nil {
		t.Errorf("answered %v, want a null result", replies[1])
	}
}
