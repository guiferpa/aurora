package main

import (
	"fmt"
	"io"
	"os"
)

type w struct{}

func (w *w) Write(bs []byte) (int, error) {
	return fmt.Fprintf(os.Stdout, "%s\n", bs)
}

func ToMainWriter() io.Writer {
	return &w{}
}
