//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"syscall/js"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/version"
)

var (
	document js.Value
	eval     func() js.Func
)

func init() {
	document = js.Global().Get("document")

	errorWriter := ToPlaygroundErrorWriter()

	eval = func() js.Func {
		return js.FuncOf(func(this js.Value, args []js.Value) any {
			editor := js.Global().Get("editor")
			value := editor.Call("getValue").String()
			bs := bytes.NewBufferString(value)
			debug := document.Call("getElementById", "debug-mode").Get("checked").Bool()
			tokens, err := lexer.New(lexer.NewLexerOptions{
				EnableLogging: debug,
			}).GetFilledTokens(bs.Bytes())
			if err != nil {
				fmt.Println(err)
				return nil
			}
			ast, err := parser.New(parser.NewParserOptions{
				Tokens:        tokens,
				EnableLogging: debug,
			}).Parse()
			if err != nil {
				errorWriter.Write([]byte(err.Error()))
				return nil
			}
			program, err := emitter.New(emitter.NewEmitterOptions{
				EnableLogging: debug,
			}).EmitProgram(ast)
			if err != nil {
				errorWriter.Write([]byte(err.Error()))
				return nil
			}

			ev := evaluator.New(evaluator.NewEvaluatorOptions{
				EnableLogging: debug,
				EchoWriter:    ToPlaygroundWriter("echo"),
				PrintWriter:   ToPlaygroundWriter("print"),
			})

			// One top-level expression at a time, reporting its value before moving on.
			// Running everything and then walking the temp map put every print first and
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
