package main

import (
	"fmt"
	"io"
)

type w struct{}

func (w *w) Write(bs []byte) (int, error) {
	return fmt.Printf("%v\n", bs)
}

func ToMainWriter() io.Writer {
	return &w{}
}
