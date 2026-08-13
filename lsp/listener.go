package lsp

import (
	"bufio"
	"io"
	"log"

	"github.com/guiferpa/aurora/lsp/messenger"
	"github.com/guiferpa/aurora/lsp/state"
)

// maxMessageBytes caps a single JSON-RPC message. bufio.Scanner defaults to 64KB, which a
// didChange carrying a moderately sized document exceeds — and the scanner then stops
// silently, leaving the server alive but deaf.
const maxMessageBytes = 16 << 20

type MethodHandler func(l *log.Logger, s *state.State, contents []byte) any

func Listen(l *log.Logger, r io.Reader, w io.Writer, handlers map[Method]MethodHandler) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxMessageBytes)
	scanner.Split(messenger.Split)
	s := state.New()

	for scanner.Scan() {
		method, id, contents, err := messenger.Decode(scanner.Bytes())
		if err != nil {
			l.Println(err)
			break
		}
		l.Println(method, string(contents))

		// exit ends the session; shutdown only answers, per the spec.
		if method == string(MethodExit) {
			return
		}
		if method == string(MethodShutdown) {
			write(l, w, NewNullResponse(id))
			continue
		}

		h, ok := handlers[Method(method)]
		if !ok {
			// A request must always be answered or the client waits forever; a
			// notification (no id) can be dropped.
			if id != nil {
				write(l, w, NewMethodNotFoundResponse(*id, method))
			}
			continue
		}

		write(l, w, h(l, s, contents))
	}

	if err := scanner.Err(); err != nil {
		l.Println(err)
	}
}

func write(l *log.Logger, w io.Writer, msg any) {
	if _, err := messenger.Write(w, msg); err != nil {
		l.Println(err)
	}
}
