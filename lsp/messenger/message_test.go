package messenger

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func frame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func TestEncodeProducesHeaderAndBody(t *testing.T) {
	got := Encode(map[string]any{"jsonrpc": "2.0"})
	const want = "Content-Length: 18\r\n\r\n{\"jsonrpc\":\"2.0\"}"
	if !strings.HasPrefix(got, "Content-Length: ") {
		t.Fatalf("missing header: %q", got)
	}
	if !strings.Contains(got, "\r\n\r\n") {
		t.Fatalf("missing separator: %q", got)
	}
	if len(got) != len(want) {
		t.Errorf("encoded length = %d, want %d (%q)", len(got), len(want), got)
	}
}

func TestDecodeReturnsMethodAndID(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantMethod string
		wantID     *int
	}{
		{
			name:       "request carries an id",
			body:       `{"jsonrpc":"2.0","id":7,"method":"textDocument/hover"}`,
			wantMethod: "textDocument/hover",
			wantID:     intPtr(7),
		},
		{
			name:       "notification has no id",
			body:       `{"jsonrpc":"2.0","method":"textDocument/didOpen"}`,
			wantMethod: "textDocument/didOpen",
			wantID:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			method, id, contents, err := Decode([]byte(frame(tc.body)))
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if method != tc.wantMethod {
				t.Errorf("method = %q, want %q", method, tc.wantMethod)
			}
			switch {
			case tc.wantID == nil && id != nil:
				t.Errorf("id = %d, want none", *id)
			case tc.wantID != nil && id == nil:
				t.Errorf("id = none, want %d", *tc.wantID)
			case tc.wantID != nil && *id != *tc.wantID:
				t.Errorf("id = %d, want %d", *id, *tc.wantID)
			}
			if string(contents) != tc.body {
				t.Errorf("contents = %q, want %q", contents, tc.body)
			}
		})
	}
}

func TestDecodeRejectsBrokenInput(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{name: "no separator", msg: "Content-Length: 2"},
		{name: "length is not a number", msg: "Content-Length: abc\r\n\r\n{}"},
		{name: "length longer than the body", msg: "Content-Length: 999\r\n\r\n{}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := Decode([]byte(tc.msg)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestSplitWaitsForTheWholeMessage(t *testing.T) {
	body := `{"jsonrpc":"2.0","method":"exit"}`
	full := frame(body)

	// A message still arriving must not be handed over early.
	advance, token, err := Split([]byte(full[:len(full)-5]), false)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if advance != 0 || token != nil {
		t.Errorf("partial message returned advance=%d token=%q", advance, token)
	}

	advance, token, err = Split([]byte(full), false)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if advance != len(full) || string(token) != full {
		t.Errorf("advance = %d, token = %q, want the whole frame", advance, token)
	}
}

func TestWriteSkipsNil(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	if _, err := Write(buf, nil); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("wrote %q for a nil message", buf.String())
	}
}

func intPtr(v int) *int {
	return &v
}
