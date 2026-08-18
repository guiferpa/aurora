package main

import (
	"log"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/hosting/lsp/state"
	"github.com/guiferpa/aurora/hosting/lsp/textdoc"
)

// document gathers what an analysis needs about the file behind a URI. Where the width comes
// from is the server's business — it walks up to the project manifest — and this is the one
// place that knows it.
func document(uri lsp.URI, text string) textdoc.Document {
	path := textdoc.PathFromURI(uri)
	return textdoc.Document{
		Filename: path,
		Source:   text,
		TapeSize: tapeSizeFor(path),
	}
}

func (sv server) didOpen(l *log.Logger, s *state.State, contents []byte) any {
	noti, err := textdoc.ParseDidOpenNotification(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := noti.Params.TextDocument.URI
	text := noti.Params.TextDocument.Text
	s.UpdateDocument(string(uri), text)

	return textdoc.NewDiagnosticsNotification(uri, sv.textdoc.ValidateCode(document(uri, text)))
}

func (sv server) didChange(l *log.Logger, s *state.State, contents []byte) any {
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
	return textdoc.NewDiagnosticsNotification(uri, sv.textdoc.ValidateCode(document(uri, text)))
}

func (sv server) didClose(l *log.Logger, s *state.State, contents []byte) any {
	noti, err := textdoc.ParseDidCloseNotification(contents)
	if err != nil {
		l.Println(err)
		return nil
	}
	s.DeleteDocument(string(noti.Params.TextDocument.URI))
	return nil
}

func (sv server) completion(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseCompletionRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	items := sv.textdoc.CompletionItemsFor(document(uri, s.GetDocument(string(uri))), req.Params.Position, s.SnippetSupport())

	return textdoc.NewCompletionResponse(req.ID, items)
}

func (sv server) hover(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseHoverRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	info := sv.textdoc.HoverInfo(document(uri, s.GetDocument(string(uri))), req.Params.Position)
	if info == "" {
		return nil
	}

	return textdoc.NewHoverResponse(req.ID, info)
}

func (sv server) semanticTokens(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseSemanticTokensRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	data := sv.textdoc.SemanticTokensFor(s.GetDocument(string(uri)))

	return textdoc.NewSemanticTokensResponse(req.ID, data)
}
