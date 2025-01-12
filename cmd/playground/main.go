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
)

var document js.Value

func init() {
	document = js.Global().Get("document")
}

func main() {
	playground := document.Call("getElementById", "editor")
	value := playground.Get("value").String()
	bs := bytes.NewBufferString(value)
	tokens, err := lexer.GetFilledTokens(bs.Bytes())
	if err != nil {
		fmt.Println(err)
		return
	}
	ast, err := parser.New(tokens).Parse()
	if err != nil {
		fmt.Println(err)
		return
	}
	insts, err := emitter.New(ast).Emit()
	if err != nil {
		fmt.Println(err)
		return
	}
	temps, err := evaluator.New(false).Evaluate(insts)
	if err != nil {
		fmt.Println(err)
		return
	}
	s := color.New(color.FgWhite, color.Bold).Sprint("=")
	for _, temp := range temps {
		fmt.Printf("%s %s\n", s, color.New(color.FgHiYellow).Sprintf("%v", temp))
	}
}
