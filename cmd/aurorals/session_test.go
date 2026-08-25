package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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

// runSession feeds the messages to a server that reads one document and nothing around it,
// and returns every decoded reply.
func runSession(t *testing.T, messages ...string) []map[string]any {
	t.Helper()

	return session(t, func(documents *state.State) textdoc.NewSessionOptions {
		return textdoc.NewSessionOptions{Lexer: lexer.New(), Parser: parser.New()}
	}, messages...)
}

// runSessionInProject feeds them to a server wired the way main wires it, which is the one
// that reaches the files a document imports. The directory is the project it is run from,
// because that is what module names resolve against.
func runSessionInProject(t *testing.T, dir string, messages ...string) []map[string]any {
	t.Helper()
	t.Chdir(dir)

	return session(t, func(documents *state.State) textdoc.NewSessionOptions {
		return textdoc.NewSessionOptions{
			Lexer:   lexer.New(),
			Parser:  parser.New(),
			Resolve: resolveModules(documents),
		}
	}, messages...)
}

// session runs one server over the messages and decodes what it wrote.
func session(t *testing.T, options func(*state.State) textdoc.NewSessionOptions, messages ...string) []map[string]any {
	t.Helper()

	in := strings.NewReader(strings.Join(messages, ""))
	out := bytes.NewBuffer(nil)
	documents := state.New()
	sv := server{textdoc: textdoc.NewSession(options(documents))}
	lsp.Listen(log.New(io.Discard, "", 0), in, out, documents, sv.handlers())

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

// A position with no name under it is returned null rather than with nothing. A request
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

// The jump that crosses a file, through the server that reads the disk: the answer names the
// module's file by the URI the client knows it as, which is the half of the answer this side
// of the port is responsible for.
func TestSessionDefinitionCrossesIntoAModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		// The manifest is what makes this a project, which is what makes src/ the root a
		// module name resolves from.
		"aurora.toml":     "[project]\n  name = \"jump\"\n\n[profiles]\n  [profiles.main]\n    source = \"src/main.ar\"\n    binary = \"bin/main\"\n",
		"src/geometry.ar": "ident area = defer { feed(0) * feed(1); };\n",
		"src/main.ar":     "use geometry as g;\nprintd g.area(2, 3);\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	uri := "file://" + filepath.ToSlash(filepath.Join(dir, "src", "main.ar"))
	replies := runSessionInProject(t, dir,
		didOpen(uri, "use geometry as g;\nprintd g.area(2, 3);\n"),
		request(7, "textDocument/definition", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 1, "character": 11},
		}),
		exitMessage,
	)

	if len(replies) != 2 {
		t.Fatalf("expected diagnostics plus the definition reply, got %d: %v", len(replies), replies)
	}
	result := replies[1]["result"].(map[string]any)
	want := "file://" + filepath.ToSlash(filepath.Join(dir, "src", "geometry.ar"))
	if result["uri"] != want {
		t.Errorf("points at %v, want %s", result["uri"], want)
	}
	start := result["range"].(map[string]any)["start"].(map[string]any)
	if start["line"] != float64(0) || start["character"] != float64(6) {
		t.Errorf("points at %v, want line 0 character 6", start)
	}
}

// Hover over something with nothing to say — a semicolon — answers null, not silence: a
// request is answered whatever the answer is.
func TestSessionHoverOfNothingIsNull(t *testing.T) {
	uri := "file:///tmp/main.ar"
	replies := runSession(t,
		didOpen(uri, "printd 42;\n"),
		request(8, "textDocument/hover", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 9},
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

// The round trip of a rename: every edit in one file, under the URI the client knows it by.
func TestSessionRenameAnswersWithTheEdits(t *testing.T) {
	uri := "file:///tmp/main.ar"
	replies := runSession(t,
		didOpen(uri, "ident area = defer {\n  ident side = feed(0);\n  side * side;\n};\n"),
		request(9, "textDocument/rename", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 2, "character": 3},
			"newName":      "edge",
		}),
		exitMessage,
	)

	if len(replies) != 2 {
		t.Fatalf("expected diagnostics plus the rename reply, got %d: %v", len(replies), replies)
	}
	changes := replies[1]["result"].(map[string]any)["changes"].(map[string]any)
	edits, touched := changes[uri]
	if !touched || len(changes) != 1 {
		t.Fatalf("changed %v, want the open file alone", changes)
	}
	if len(edits.([]any)) != 3 {
		t.Errorf("made %d edits, want the declaration and its two readings", len(edits.([]any)))
	}
	for _, edit := range edits.([]any) {
		if edit.(map[string]any)["newText"] != "edge" {
			t.Errorf("an edit writes %v, want the new name", edit)
		}
	}
}

// A name that cannot be renamed is refused with the reason, which is what a client shows to
// whoever asked — and it is refused at the prepare step, before the box opens.
func TestSessionRenameIsRefusedWithAReason(t *testing.T) {
	uri := "file:///tmp/main.ar"
	replies := runSession(t,
		didOpen(uri, "ident total = 1;\nprintd total;\n"),
		request(10, "textDocument/prepareRename", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 1, "character": 8},
		}),
		exitMessage,
	)

	if len(replies) != 2 {
		t.Fatalf("expected diagnostics plus the refusal, got %d: %v", len(replies), replies)
	}
	failure, refused := replies[1]["error"].(map[string]any)
	if !refused {
		t.Fatalf("answered %v, want a refusal", replies[1])
	}
	if !strings.Contains(fmt.Sprint(failure["message"]), "another file may be importing it") {
		t.Errorf("said %v, want it to say why", failure["message"])
	}
}
