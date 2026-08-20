package main

import (
	"log"
	"path/filepath"

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
		// Nothing to say is a null result rather than silence, for the same reason a jump
		// that lands nowhere is: the request carries an id and the client waits on it.
		return lsp.NewNullResponse(&req.ID)
	}

	return textdoc.NewHoverResponse(req.ID, info)
}

// definition answers where the name under the cursor was declared.
//
// The path comes back from the session as a path, because which URI a file has is the
// client's vocabulary and not the compiler's. Turning it back into one is this side's job,
// and it is done from the absolute path so a module named from the source root — src/x.ar —
// is the same file the editor already has open under its own URI.
func (sv server) definition(l *log.Logger, s *state.State, contents []byte) any {
	req, err := textdoc.ParseDefinitionRequest(contents)
	if err != nil {
		l.Println(err)
		return nil
	}

	uri := req.Params.TextDocument.URI
	found, ok := sv.textdoc.DefinitionFor(document(uri, s.GetDocument(string(uri))), req.Params.Position)
	if !ok {
		// A request is answered, always. Nothing under the cursor is a null result rather
		// than silence: silence leaves the client waiting for a reply that never comes.
		return lsp.NewNullResponse(&req.ID)
	}

	return textdoc.NewDefinitionResponse(req.ID, lsp.Location{URI: uriOf(found.Filename), Range: found.Range})
}

// uriOf turns a path into the URI the client knows the file by.
func uriOf(path string) lsp.URI {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	return lsp.URI("file://" + filepath.ToSlash(absolute))
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
