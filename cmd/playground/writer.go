//go:build js && wasm

package main

import (
	"io"
	"syscall/js"
)

type pw struct {
	builtin string
}

func (w *pw) Write(bs []byte) (int, error) {
	u8 := js.Global().Get("Uint8Array").New(len(bs))
	js.CopyBytesToJS(u8, bs)
	js.Global().Call("evalResultHandler", u8, w.builtin)
	return len(bs), nil
}

type pew struct{}

func (w *pew) Write(bs []byte) (int, error) {
	u8 := js.Global().Get("Uint8Array").New(len(bs))
	js.CopyBytesToJS(u8, bs)
	js.Global().Call("evalErrorHandler", u8)
	return len(bs), nil
}

func ToPlaygroundWriter(builtin string) io.Writer {
	return &pw{builtin}
}

func ToPlaygroundErrorWriter() io.Writer {
	return &pew{}
}
