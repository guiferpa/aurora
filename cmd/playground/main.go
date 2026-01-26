//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"syscall/js"

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
			tokens, err := lexer.New(lexer.NewLexerOptions{
				EnableLogging: false,
			}).GetFilledTokens(bs.Bytes())
			if err != nil {
				fmt.Println(err)
				return nil
			}
			ast, err := parser.New(tokens, parser.NewParserOptions{
				Filename:      "",
				EnableLogging: false,
			}).Parse()
			if err != nil {
				errorWriter.Write([]byte(err.Error()))
				return nil
			}
			insts, err := emitter.New(emitter.NewEmitterOptions{
				EnableLogging: false,
			}).Emit(ast)
			if err != nil {
				errorWriter.Write([]byte(err.Error()))
				return nil
			}

			temps, err := evaluator.New(evaluator.NewEvaluatorOptions{
				EnableLogging: false,
				EchoWriter:    ToPlaygroundWriter("echo"),
				PrintWriter:   ToPlaygroundWriter("print"),
			}).Evaluate(insts)
			if err != nil {
				errorWriter.Write([]byte(err.Error()))
				return nil
			}
			for _, temp := range temps {
				u8 := js.Global().Get("Uint8Array").New(len(temp))
				js.CopyBytesToJS(u8, temp)
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
