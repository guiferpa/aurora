package main

import (
	"fmt"
	"io"
)

// print and echo are two builtins with two purposes: print shows the raw bytes of a value,
// echo reads those bytes back as text. The evaluator takes a writer for each, so the two
// live here rather than sharing one.

type printWriter struct{}

func (w *printWriter) Write(bs []byte) (int, error) {
	return fmt.Printf("%v\n", bs)
}

type echoWriter struct{}

func (w *echoWriter) Write(bs []byte) (int, error) {
	return fmt.Printf("%s\n", bs)
}

// ToMainWriter returns the writer for print: raw bytes.
func ToMainWriter() io.Writer {
	return &printWriter{}
}

// ToEchoWriter returns the writer for echo: bytes rendered as text.
func ToEchoWriter() io.Writer {
	return &echoWriter{}
}
