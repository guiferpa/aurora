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

var document js.Value

func init() {
	document = js.Global().Get("document")
}

func main() {
	eval := func() js.Func {
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
				fmt.Println(err)
				return nil
			}
			insts, err := emitter.New(emitter.NewEmitterOptions{
				EnableLogging: false,
			}).Emit(ast)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			temps, err := evaluator.New(evaluator.NewEvaluatorOptions{
				EnableLogging: false,
			}).Evaluate(insts)
			if err != nil {
				fmt.Println(err)
				return nil
			}
			output := document.Call("getElementById", "output")
			for _, temp := range temps {
				li := document.Call("createElement", "li")
				li.Set("innerHTML", fmt.Sprintf("= %v", temp))
				output.Call("appendChild", li)
			}
			return nil
		})
	}
	evalrunner := eval()
	defer evalrunner.Release()

	document.Call("getElementById", "version").Set("innerText", fmt.Sprintf("Aurora version: %s", version.VERSION))
	document.Call("getElementById", "runner").Call("addEventListener", "click", evalrunner)

	select {}
}
