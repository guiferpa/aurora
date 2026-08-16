//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"os"
	"syscall/js"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/internal/trace"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/version"
)

// defaultTapeSize is the width the playground starts at, and it is wider than the language's
// default of 8 on purpose.
//
// Someone meeting Aurora here writes text before they write anything else, and the page opens
// on printc "Hello, World!" — thirteen bytes, which does not fit an eight-byte tape. Being
// told so as a first experience teaches nothing about tapes and everything about giving up.
// The control says what the width is, so the number is on screen rather than assumed.
const defaultTapeSize = 32

var (
	document js.Value
	eval     func() js.Func
)

// tapeSize reads the width the page is set to. Anything the page cannot answer for — no
// control, a value that is not a number, a number outside what a tape can be — falls back to
// the playground's own default rather than to the language's.
func tapeSize() int {
	el := document.Call("getElementById", "tape-size")
	if !el.Truthy() {
		return defaultTapeSize
	}
	return byteutil.ParseTapeSize(el.Get("value").String(), defaultTapeSize)
}

func init() {
	document = js.Global().Get("document")

	errorWriter := ToPlaygroundErrorWriter()

	eval = func() js.Func {
		return js.FuncOf(func(this js.Value, args []js.Value) any {
			editor := js.Global().Get("editor")
			value := editor.Call("getValue").String()
			bs := bytes.NewBufferString(value)
			debug := document.Call("getElementById", "debug-mode").Get("checked").Bool()
			// Read for every run, so changing the width is a matter of running again.
			size := tapeSize()
			tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens(bs.Bytes())
			if err != nil {
				fmt.Println(err)
				return nil
			}
			if debug {
				if err := trace.Tokens(os.Stdout, tokens); err != nil {
					errorWriter.Write([]byte(err.Error()))
					return nil
				}
			}
			ast, err := parser.New(parser.NewParserOptions{
				Tokens:   tokens,
				TapeSize: size,
			}).Parse()
			if err != nil {
				errorWriter.Write([]byte(err.Error()))
				return nil
			}
			// The phases return what they made; the page decides to show it. In wasm this
			// lands in the browser console, which is where the debug checkbox points.
			if debug {
				if err := trace.AST(os.Stdout, ast); err != nil {
					errorWriter.Write([]byte(err.Error()))
					return nil
				}
			}
			program, err := emitter.New(emitter.NewEmitterOptions{
				TapeSize: size,
			}).EmitProgram(ast)
			if err != nil {
				errorWriter.Write([]byte(err.Error()))
				return nil
			}
			if debug {
				if err := trace.Instructions(os.Stdout, program.Instructions); err != nil {
					errorWriter.Write([]byte(err.Error()))
					return nil
				}
			}

			ev := evaluator.New(evaluator.NewEvaluatorOptions{
				// The print builtins each format their own reading of a tape, so what
				// arrives here is the finished line and the page only has to show it.
				Output:   ToPlaygroundWriter("output"),
				TapeSize: size,
			})

			// One top-level expression at a time, reporting its value before moving on.
			// Running everything and then walking the temp map put every printed line first and
			// the values after, in no order at all — a map has none.
			for _, expr := range program.Expressions {
				temps, err := ev.EvaluateRange(program.Instructions, uint64(expr.From), uint64(expr.To))
				if err != nil {
					errorWriter.Write([]byte(err.Error()))
					return nil
				}
				value, ok := temps[byteutil.ToHex(expr.Label)]
				if !ok {
					continue
				}
				u8 := js.Global().Get("Uint8Array").New(len(value))
				js.CopyBytesToJS(u8, value)
				js.Global().Call("evalResultHandler", u8)
			}
			return nil
		})
	}
}

func main() {
	evalrunner := eval()
	defer evalrunner.Release()

	document.Call("getElementById", "version").Set("innerText", fmt.Sprintf("Aurora version: %s", version.VERSION))
	document.Call("getElementById", "runner").Call("addEventListener", "click", evalrunner)

	select {}
}
