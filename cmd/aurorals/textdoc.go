package main

import (
	"fmt"
	"log"
	"strings"

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

	items := []textdoc.CompletionItem{
		{
			Label:         "Golang",
			Detail:        "Simple and fast programming laguage",
			Documentation: "Go is expressive, concise, clean, and efficient.\nIt's a fast, statically typed, compiled language.",
		},
	}

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

	diagnocstics := textdoc.Diagnocstics{}

	return textdoc.NewDiagnosticsNotification(uri, diagnocstics)
}

func TextdocDidChangeHandler(l *log.Logger, s *state.State, contents []byte) any {
	noti, err := textdoc.ParseDidChangeNotification(contents)
	if err != nil {
		l.Println(err)
		return nil

	}

	uri := noti.Params.TextDocument.URI
	for _, changes := range noti.Params.ContentChanges {
		s.UpdateDocument(string(uri), changes.Text)
	}

	diagnocstics := textdoc.Diagnocstics{}

	return textdoc.NewDiagnosticsNotification(uri, diagnocstics)
}

func TextdocHoverHandler(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseHoverRequest(contents)
	if err != nil {
		l.Println(err)
		return nil

	}

	uri := req.Params.TextDocument.URI
	doc := s.GetDocument(string(uri))

	return textdoc.NewHoverResponse(req.ID, fmt.Sprintf("Doc: %s, Chars: %d", uri, len(doc)))
}

func TextdocDefinitionHandler(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseDefinitionRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	position := req.Params.Position

	return textdoc.NewDefinitionResponse(req.ID, uri, position)
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
