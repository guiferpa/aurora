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
	eval := func(debug bool) js.Func {
		return js.FuncOf(func(this js.Value, args []js.Value) any {
			playground := document.Call("getElementById", "editor")
			value := playground.Get("value").String()
			bs := bytes.NewBufferString(value)
			tokens, err := lexer.GetFilledTokens(bs.Bytes())
			if err != nil {
				fmt.Println(err)
				return nil
			}
			ast, err := parser.New(tokens).Parse()
			if err != nil {
				fmt.Println(err)
				return nil
			}
			insts, err := emitter.New().Emit(ast)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			emitter.Print(insts, debug)
			temps, err := evaluator.New(debug).Evaluate(insts)
			if err != nil {
				fmt.Println(err)
				return nil
			}
			preview := document.Call("getElementById", "preview")
			for _, temp := range temps {
				li := document.Call("createElement", "li")
				li.Set("innerHTML", fmt.Sprintf("= %v", temp))
				preview.Call("appendChild", li)
			}
			return nil
		})
	}
	evalrunner := eval(false)
	defer evalrunner.Release()

	evaldebugger := eval(true)
	defer evaldebugger.Release()

	document.Call("getElementById", "version").Set("innerText", fmt.Sprintf("Aurora version: %s", version.VERSION))
	document.Call("getElementById", "runner").Call("addEventListener", "click", evalrunner)
	document.Call("getElementById", "debugger").Call("addEventListener", "click", evaldebugger)

	select {}
}
