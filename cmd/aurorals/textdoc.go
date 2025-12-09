package main

import (
	"log"
	"strings"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/lsp"
	"github.com/guiferpa/aurora/lsp/state"
	"github.com/guiferpa/aurora/lsp/textdoc"
)

func TextdocCompletionHandler(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseCompletionRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	items := make([]textdoc.CompletionItem, 0)
	tags := lexer.GetProcessableTags()
	for _, t := range tags {
		items = append(items, textdoc.CompletionItem{
			Label:  t.Keyword,
			Detail: t.Description,
			Kind:   textdoc.Keyword,
		})
	}

	l.Println(items)

	return textdoc.NewCompletionResponse(req.ID, items)
}

func TextdocDidOpenHandler(l *log.Logger, s *state.State, contents []byte) any {
	noti, err := textdoc.ParseDidOpenNotification(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := noti.Params.TextDocument.URI
	text := noti.Params.TextDocument.Text
	s.UpdateDocument(string(uri), text)

	// Validate the code and generate diagnostics
	diagnostics := textdoc.ValidateCode(text)

	return textdoc.NewDiagnosticsNotification(uri, diagnostics)
}

func TextdocDidChangeHandler(l *log.Logger, s *state.State, contents []byte) any {
	noti, err := textdoc.ParseDidChangeNotification(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := noti.Params.TextDocument.URI
	var updatedText string
	for _, changes := range noti.Params.ContentChanges {
		updatedText = changes.Text
		s.UpdateDocument(string(uri), updatedText)
	}

	// Validate the updated code and generate diagnostics
	diagnostics := textdoc.ValidateCode(updatedText)

	return textdoc.NewDiagnosticsNotification(uri, diagnostics)
}

func TextdocHoverHandler(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseHoverRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	doc := s.GetDocument(string(uri))
	pos := req.Params.Position

	// Get hover information for the position
	hoverInfo := textdoc.GetHoverInfo(doc, pos)
	if hoverInfo == "" {
		return nil
	}

	return textdoc.NewHoverResponse(req.ID, hoverInfo)
}

func TextdocDefinitionHandler(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseDefinitionRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	doc := s.GetDocument(string(uri))
	pos := req.Params.Position

	// Get the token at the position
	token, err := textdoc.GetTokenAtPosition(doc, pos)
	if err != nil || token == nil {
		return nil
	}

	// Only handle identifier definitions
	if token.GetTag().Id != lexer.ID {
		return nil
	}

	// Find the definition
	identName := string(token.GetMatch())
	defPos, found := textdoc.FindIdentifierDefinition(doc, identName)
	if !found {
		return nil
	}

	return textdoc.NewDefinitionResponse(req.ID, uri, defPos)
}

func TextdocCodeActionHandler(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseCodeActionRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	doc := s.GetDocument(string(uri))
	toReplace := "Java"
	cas := textdoc.CodeActions{}

	for row, line := range strings.Split(doc, "\n") {
		idx := strings.Index(line, toReplace)
		if idx >= 0 {
			// ----- 1. replace text action -------
			replaceChange := map[lsp.URI][]textdoc.TextEdit{}
			replaceChange[uri] = []textdoc.TextEdit{
				{
					Range:   lsp.LineRange(row, idx, idx+len(toReplace)),
					NewText: "Golang",
				},
			}

			cas = append(cas, textdoc.CodeAction{
				Title: "Replace Ja*a with a superior language",
				Edit:  &textdoc.WorkspaceEdit{Changes: replaceChange},
			})

			// ----- 2. censor text action -------
			censorChange := map[lsp.URI][]textdoc.TextEdit{}
			censorChange[uri] = []textdoc.TextEdit{
				{
					Range:   lsp.LineRange(row, idx, idx+len(toReplace)),
					NewText: "Ja*a",
				},
			}

			cas = append(cas, textdoc.CodeAction{
				Title: "Censor to Ja*a",
				Edit:  &textdoc.WorkspaceEdit{Changes: censorChange},
			})
		}
	}

	return textdoc.NewCodeActionResponse(req.ID, cas)
}
