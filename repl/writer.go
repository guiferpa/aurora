package repl

import (
	"fmt"
)

type EchoWriter struct{}

func (w *EchoWriter) Write(bs []byte) (int, error) {
	return fmt.Printf("%s\n", bs)
}

type PrintWriter struct{}

func (w *PrintWriter) Write(bs []byte) (int, error) {
	return fmt.Printf("%v\n", bs)
}
