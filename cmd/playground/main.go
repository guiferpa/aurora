//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"syscall/js"

	"github.com/fatih/color"
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
			insts, err := emitter.New(ast).Emit()
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
			s := color.New(color.FgWhite, color.Bold).Sprint("=")
			for _, temp := range temps {
				fmt.Printf("%s %s\n", s, color.New(color.FgHiYellow).Sprintf("%v", temp))
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
