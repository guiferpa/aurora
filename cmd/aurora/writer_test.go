package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// capture runs f with stdout redirected, returning what was printed. The writers print
// directly rather than taking a stream, so this is the only way to see them.
func capture(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = saved }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	f()
	_ = w.Close()
	return <-done
}

// print and echo are two builtins with two purposes: print shows the bytes of a value, echo
// reads those bytes back as text. They shared a writer once, and echo printed byte arrays.
func TestWritersRenderDifferently(t *testing.T) {
	value := []byte{104, 105} // "hi"

	printed := capture(t, func() {
		_, _ = ToMainWriter().Write(value)
	})
	if want := "[104 105]"; !strings.Contains(printed, want) {
		t.Errorf("print wrote %q, want it to contain %q", printed, want)
	}

	echoed := capture(t, func() {
		_, _ = ToEchoWriter().Write(value)
	})
	if want := "hi"; !strings.Contains(echoed, want) {
		t.Errorf("echo wrote %q, want it to contain %q", echoed, want)
	}
	if strings.Contains(echoed, "[") {
		t.Errorf("echo wrote %q, which looks like raw bytes", echoed)
	}
}

func TestWritersEndWithANewline(t *testing.T) {
	for _, tc := range []struct {
		name   string
		writer io.Writer
	}{
		{name: "print", writer: ToMainWriter()},
		{name: "echo", writer: ToEchoWriter()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := capture(t, func() {
				_, _ = tc.writer.Write([]byte{65})
			})
			if !strings.HasSuffix(got, "\n") {
				t.Errorf("wrote %q, want it to end in a newline", got)
			}
		})
	}
}
