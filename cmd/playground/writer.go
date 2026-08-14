//go:build js && wasm

package main

import (
	"io"
	"syscall/js"
)

// pw carries what a program printed to the page. The kind tells the page whether the bytes
// are a finished line to show as it is, or a value it has to render itself.
type pw struct {
	kind string
}

func (w *pw) Write(bs []byte) (int, error) {
	u8 := js.Global().Get("Uint8Array").New(len(bs))
	js.CopyBytesToJS(u8, bs)
	js.Global().Call("evalResultHandler", u8, w.kind)
	return len(bs), nil
}

type pew struct{}

func (w *pew) Write(bs []byte) (int, error) {
	u8 := js.Global().Get("Uint8Array").New(len(bs))
	js.CopyBytesToJS(u8, bs)
	js.Global().Call("evalErrorHandler", u8)
	return len(bs), nil
}

func ToPlaygroundWriter(kind string) io.Writer {
	return &pw{kind}
}

func ToPlaygroundErrorWriter() io.Writer {
	return &pew{}
}
