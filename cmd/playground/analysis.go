//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/hosting/lsp/textdoc"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

// The page is a third client of the analyses the language server answers with, next to the
// editor plugin and the CLI's own use of the phases: the colours come from the lexer and the
// marks from the parser.
//
// Writing them again in JavaScript — a Monarch grammar, a list of keywords — is the drift
// this avoids. A keyword added to the language shows up here without anyone remembering to
// add it twice.

// playgroundFile is the name the page's document goes by. It decides file-scoped rules, and
// it is deliberately not a test file: assert stays refused here, as it was before the page
// analysed anything at all.
const playgroundFile = "playground.ar"

// documentFrom gathers what an analysis reads out of the call from the page. The width comes
// from the control, which is how the marks agree with what Run does: the same number reaches
// both.
func documentFrom(args []js.Value) textdoc.Document {
	source := ""
	if len(args) > 0 {
		source = args[0].String()
	}
	return textdoc.Document{
		Filename: playgroundFile,
		Source:   source,
		TapeSize: tapeSize(),
	}
}

// positionFrom reads the place in the document a call is asking about. Both counts start at
// zero here, as they do in the protocol; the page is what converts, since it is the editor
// that counts from one.
func positionFrom(args []js.Value) lsp.Position {
	pos := lsp.Position{}
	if len(args) > 1 {
		pos.Line = args[1].Int()
	}
	if len(args) > 2 {
		pos.Character = args[2].Int()
	}
	return pos
}

// marshal answers with JSON, which is what crosses to the page. A value that cannot be
// encoded answers as nothing rather than as a broken string: an editor that colours nothing
// is a worse day than one that colours late, and neither is worth stopping the page for.
func marshal(v any) any {
	bs, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return string(bs)
}

// analyses registers what the page needs to mark and colour a document.
//
// Each takes the source as it stands and answers from scratch: a document being typed is
// small, and the alternative — telling the page how to describe a change — is a protocol,
// which is what the editor plugin has and the page does not need.
func analyses() []js.Func {
	// One session for the page, the way the language server holds one for an editor: the
	// phases are built here and the width comes with each document.
	session := textdoc.NewSession(textdoc.NewSessionOptions{
		Lexer:  lexer.New(),
		Parser: parser.New(),
	})

	diagnostics := js.FuncOf(func(this js.Value, args []js.Value) any {
		return marshal(session.ValidateCode(documentFrom(args)))
	})

	semanticTokens := js.FuncOf(func(this js.Value, args []js.Value) any {
		return marshal(session.SemanticTokensFor(documentFrom(args).Source))
	})

	hover := js.FuncOf(func(this js.Value, args []js.Value) any {
		return marshal(session.HoverInfo(documentFrom(args), positionFrom(args)))
	})

	// Snippets are always on: the page is one editor and it is known to expand them, where a
	// language server has to ask because it does not know who is listening.
	completions := js.FuncOf(func(this js.Value, args []js.Value) any {
		return marshal(session.CompletionItemsFor(documentFrom(args), positionFrom(args), true))
	})

	// The legend names what the numbers in the token data mean, and the page has to be given
	// it rather than told to keep a copy: the order is the wire format.
	semanticLegend := js.FuncOf(func(this js.Value, args []js.Value) any {
		return marshal(map[string]any{
			"tokenTypes":     textdoc.SemanticTokenTypes,
			"tokenModifiers": textdoc.SemanticTokenModifiers,
		})
	})

	js.Global().Set("auroraDiagnostics", diagnostics)
	js.Global().Set("auroraSemanticTokens", semanticTokens)
	js.Global().Set("auroraSemanticLegend", semanticLegend)
	js.Global().Set("auroraHover", hover)
	js.Global().Set("auroraCompletions", completions)

	return []js.Func{diagnostics, semanticTokens, semanticLegend, hover, completions}
}
