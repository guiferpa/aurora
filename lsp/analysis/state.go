package analysis

import (
	"fmt"
	"strings"

	"github.com/guiferpa/aurora/lsp"
	"github.com/guiferpa/aurora/lsp/textdoc"
)

// saves current state of all opened documents
type State struct {
	// map of filenames to content
	Documents map[string]string
}

func NewState() State {
	return State{Documents: map[string]string{}}
}

func getDiagnosticsForFile(text string) []textdoc.Diagnostic {
	diagnostics := []textdoc.Diagnostic{}
	for row, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "Java") {
			idx := strings.Index(line, "Java")
			diagnostics = append(diagnostics, textdoc.Diagnostic{
				Range:    LineRange(row, idx, idx+len("Java")),
				Severity: 1, // error
				Source:   "GoLSP",
				Message:  "Dude! Mind your language!",
			})
		}

		if strings.Contains(line, "Golang") {
			idx := strings.Index(line, "Golang")
			diagnostics = append(diagnostics, textdoc.Diagnostic{
				Range:    LineRange(row, idx, idx+len("Golang")),
				Severity: 4, // hint
				Source:   "GoLSP",
				Message:  "Great choice!",
			})
		}
	}
	return diagnostics
}

func (s *State) OpenDocument(uri, text string) []textdoc.Diagnostic {
	s.Documents[uri] = text
	return getDiagnosticsForFile(text)
}

func (s *State) UpdateDocument(uri, text string) []textdoc.Diagnostic {
	s.Documents[uri] = text
	return getDiagnosticsForFile(text)
}

func (s *State) Hover(id int, uri string, position lsp.Position) lsp.HoverResponse {
	document := s.Documents[uri]
	return lsp.HoverResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: lsp.HoverResult{
			Contents: fmt.Sprintf("Doc: %s Characters: %d", uri, len(document)),
		},
	}
}

func (s *State) Definition(id int, uri string, position lsp.Position) textdoc.DefinitionResponse {
	return textdoc.DefinitionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: lsp.Location{
			// we are saying that definition is in same document
			URI: uri,
			Range: lsp.Range{
				// just for the demo, whatever definition we ask for, we always return first character on 1 line above the cursor
				// so that we know that the cursor is doing something
				// in  real life we would look up the definition in another file
				Start: lsp.Position{
					Line:      position.Line - 1,
					Character: 0,
				},
				End: lsp.Position{
					Line:      position.Line - 1,
					Character: 0,
				},
			},
		},
	}
}

func (s *State) TextDocumentCodeAction(id int, uri string) textdoc.CodeActionResponse {
	text := s.Documents[uri]
	toReplace := "Java"
	actions := []textdoc.CodeAction{}

	// process each line
	for row, line := range strings.Split(text, "\n") {
		idx := strings.Index(line, toReplace)
		if idx >= 0 { // means string present

			// ----- 1. replace text action -------
			replaceChange := map[string][]textdoc.TextEdit{}
			replaceChange[uri] = []textdoc.TextEdit{
				{
					Range:   LineRange(row, idx, idx+len(toReplace)),
					NewText: "Golang",
				},
			}

			actions = append(actions, textdoc.CodeAction{
				Title: "Replace Ja*a with a superior language",
				Edit:  &textdoc.WorkspaceEdit{Changes: replaceChange},
			})

			// ----- 2. censor text action -------
			censorChange := map[string][]textdoc.TextEdit{}
			censorChange[uri] = []textdoc.TextEdit{
				{
					Range:   LineRange(row, idx, idx+len(toReplace)),
					NewText: "Ja*a",
				},
			}

			actions = append(actions, textdoc.CodeAction{
				Title: "Censor to Ja*a",
				Edit:  &textdoc.WorkspaceEdit{Changes: censorChange},
			})
		}
	}

	return textdoc.CodeActionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: actions,
	}
}

func (s *State) TextDocumentCompletion(id int, uri string) textdoc.CompletionResponse {
	// we can do things like asking static analysis tools to figure out your completion

	// when we type 'Go', the text editor will show these suggestions.
	items := []textdoc.CompletionItem{
		{
			Label:         "Golang",
			Detail:        "Simple and fast programming laguage",
			Documentation: "Go is expressive, concise, clean, and efficient.\nIt's a fast, statically typed, compiled language.",
		},
	}
	return textdoc.CompletionResponse{
		Response: lsp.Response{
			RPC: "2.0",
			ID:  &id,
		},
		Result: textdoc.CompletionResult{
			Items: items,
		},
	}
}

func LineRange(line, start, end int) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{
			Line:      line,
			Character: start,
		},
		End: lsp.Position{
			Line:      line,
			Character: end,
		},
	}
}
