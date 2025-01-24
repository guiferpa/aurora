package lsp

import (
	"bufio"
	"io"
	"log"

	"github.com/guiferpa/aurora/lsp/messenger"
	"github.com/guiferpa/aurora/lsp/state"
)

type MethodHandler func(l *log.Logger, s *state.State, contents []byte) any

func Listen(l *log.Logger, r io.Reader, w io.Writer, handlers map[Method]MethodHandler) {
	scanner := bufio.NewScanner(r)
	scanner.Split(messenger.Split)
	s := state.New()

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := messenger.Decode(msg)
		if err != nil {
			l.Println(err)
			break
		}
		l.Println(method, string(contents))
		h, ok := handlers[Method(method)]
		if !ok {
			continue
		}
		if _, err := messenger.Write(w, h(l, s, contents)); err != nil {
			l.Println(err)
			continue
		}
	}
}
