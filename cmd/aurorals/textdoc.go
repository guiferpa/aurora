package main

import (
	"log"

	"github.com/guiferpa/aurora/lsp/state"
	"github.com/guiferpa/aurora/lsp/textdoc"
)

func TextdocDidOpenHandler(l *log.Logger, s *state.State, contents []byte) any {
	noti, err := textdoc.ParseDidOpenNotification(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := noti.Params.TextDocument.URI
	text := noti.Params.TextDocument.Text
	s.UpdateDocument(string(uri), text)

	return textdoc.NewDiagnosticsNotification(uri, textdoc.ValidateCode(textdoc.PathFromURI(uri), text))
}

func TextdocDidChangeHandler(l *log.Logger, s *state.State, contents []byte) any {
	noti, err := textdoc.ParseDidChangeNotification(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := noti.Params.TextDocument.URI
	// Sync is full, so the last change event carries the whole document.
	text := s.GetDocument(string(uri))
	for _, change := range noti.Params.ContentChanges {
		text = change.Text
	}
	s.UpdateDocument(string(uri), text)

	// Published on every change, including when it comes back empty: that is what clears
	// an error the user just fixed.
	return textdoc.NewDiagnosticsNotification(uri, textdoc.ValidateCode(textdoc.PathFromURI(uri), text))
}

func TextdocDidCloseHandler(l *log.Logger, s *state.State, contents []byte) any {
	noti, err := textdoc.ParseDidCloseNotification(contents)
	if err != nil {
		l.Println(err)
		return nil
	}
	s.DeleteDocument(string(noti.Params.TextDocument.URI))
	return nil
}

func TextdocCompletionHandler(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseCompletionRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	items := textdoc.CompletionItemsFor(textdoc.PathFromURI(uri), s.GetDocument(string(uri)), req.Params.Position, s.SnippetSupport())

	return textdoc.NewCompletionResponse(req.ID, items)
}

func TextdocHoverHandler(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseHoverRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	info := textdoc.HoverInfo(textdoc.PathFromURI(uri), s.GetDocument(string(uri)), req.Params.Position)
	if info == "" {
		return nil
	}

	return textdoc.NewHoverResponse(req.ID, info)
}

func TextdocSemanticTokensHandler(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseSemanticTokensRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	data := textdoc.SemanticTokensFor(s.GetDocument(string(uri)))

	return textdoc.NewSemanticTokensResponse(req.ID, data)
}
