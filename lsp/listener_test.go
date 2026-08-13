package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/lsp/state"
)

func frame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func discardLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestListenDispatchesToHandler(t *testing.T) {
	in := strings.NewReader(frame(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`) +
		frame(`{"jsonrpc":"2.0","method":"exit"}`))
	out := bytes.NewBuffer(nil)

	called := 0
	Listen(discardLogger(), in, out, map[Method]MethodHandler{
		"initialize": func(l *log.Logger, s *state.State, contents []byte) any {
			called++
			return NullResponse{Response: Response{RPC: "2.0", ID: intPtr(1)}}
		},
	})

	if called != 1 {
		t.Errorf("handler called %d times, want 1", called)
	}
	if !strings.Contains(out.String(), `"id":1`) {
		t.Errorf("no response written: %q", out.String())
	}
}

// A request the server does not implement must still get an answer, or the client blocks
// waiting for a reply that never comes.
func TestListenAnswersUnknownRequest(t *testing.T) {
	in := strings.NewReader(frame(`{"jsonrpc":"2.0","id":9,"method":"textDocument/formatting"}`) +
		frame(`{"jsonrpc":"2.0","method":"exit"}`))
	out := bytes.NewBuffer(nil)

	Listen(discardLogger(), in, out, map[Method]MethodHandler{})

	body := out.String()
	if !strings.Contains(body, fmt.Sprint(CodeMethodNotFound)) {
		t.Errorf("expected a method-not-found error, got %q", body)
	}
	if !strings.Contains(body, `"id":9`) {
		t.Errorf("error response must carry the request id, got %q", body)
	}
}

func TestListenIgnoresUnknownNotification(t *testing.T) {
	in := strings.NewReader(frame(`{"jsonrpc":"2.0","method":"$/setTrace"}`) +
		frame(`{"jsonrpc":"2.0","method":"exit"}`))
	out := bytes.NewBuffer(nil)

	Listen(discardLogger(), in, out, map[Method]MethodHandler{})

	if out.Len() != 0 {
		t.Errorf("a notification needs no answer, got %q", out.String())
	}
}

func TestListenAnswersShutdownAndStopsOnExit(t *testing.T) {
	in := strings.NewReader(frame(`{"jsonrpc":"2.0","id":3,"method":"shutdown"}`) +
		frame(`{"jsonrpc":"2.0","method":"exit"}`) +
		frame(`{"jsonrpc":"2.0","id":4,"method":"initialize"}`))
	out := bytes.NewBuffer(nil)

	called := 0
	Listen(discardLogger(), in, out, map[Method]MethodHandler{
		"initialize": func(l *log.Logger, s *state.State, contents []byte) any {
			called++
			return nil
		},
	})

	if !strings.Contains(out.String(), `"result":null`) {
		t.Errorf("shutdown must be answered with a null result, got %q", out.String())
	}
	if called != 0 {
		t.Error("nothing may be handled after exit")
	}
}

// bufio.Scanner defaults to a 64KB token; a didChange carrying a real document exceeds it
// and the server would go deaf without a word.
func TestListenHandlesMessageLargerThan64KB(t *testing.T) {
	text := strings.Repeat("ident a = 1;\n", 12000) // ~150KB
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "textDocument/didChange",
		"params":  map[string]any{"text": text},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(body) < 64*1024 {
		t.Fatalf("test payload is only %d bytes, it must exceed the scanner default", len(body))
	}

	in := strings.NewReader(frame(string(body)) + frame(`{"jsonrpc":"2.0","method":"exit"}`))
	got := 0
	Listen(discardLogger(), in, bytes.NewBuffer(nil), map[Method]MethodHandler{
		"textDocument/didChange": func(l *log.Logger, s *state.State, contents []byte) any {
			got = len(contents)
			return nil
		},
	})

	if got != len(body) {
		t.Errorf("handler received %d bytes, want %d", got, len(body))
	}
}

func intPtr(v int) *int {
	return &v
}
