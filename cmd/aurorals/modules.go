package main

import (
	"os"
	"path/filepath"

	"github.com/guiferpa/aurora/hosting/lsp"
	"github.com/guiferpa/aurora/hosting/lsp/state"
	"github.com/guiferpa/aurora/hosting/lsp/textdoc"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/resolver"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
)

// How the language server reaches the files a document imports.
//
// This is the third reader of the same port. The command line reads a disk; the playground
// reads a map it already holds, since there is no file system in a browser; and this one
// reads the editor's buffers first and the disk only for what is not open. An editor
// answering from the disk would be answering about a version the person editing has already
// moved past — a name they just wrote would be underlined as missing, and one they just
// deleted would go on resolving.

// resolveModules answers with the modules a document imports, for a tree the server already
// parsed. It is the port textdoc is handed: everything about the world is on this side of it.
func resolveModules(s *state.State) textdoc.Resolve {
	lx := lexer.New()
	ps := parser.New()

	return func(doc textdoc.Document, uses []ast.UseDeclaration) ([]module.Module, error) {
		return resolver.New(resolver.Options{
			SourceRoot: sourceRootFor(doc.Filename),
			Read:       readThroughBuffers(s),
			Parse: func(filename string, id module.ID, source []byte, imports map[string]ast.Offer) (ast.AST, error) {
				tokens, err := lx.GetFilledTokens(source)
				if err != nil {
					return ast.AST{}, err
				}
				return ps.Parse(parser.ParseInput{
					Filename: filename,
					Tokens:   tokens,
					TapeSize: doc.TapeSize,
					Module:   string(id),
					Imports:  imports,
				})
			},
			Header: func(source []byte) ([]ast.UseDeclaration, error) {
				tokens, err := lx.GetFilledTokens(source)
				if err != nil {
					return nil, err
				}
				return parser.ScanUses(tokens), nil
			},
		}).DependenciesOf(doc.Filename, uses)
	}
}

// readThroughBuffers answers with what the editor is showing, and falls back to the disk.
func readThroughBuffers(s *state.State) resolver.Read {
	return func(path string) ([]byte, error) {
		if text, open := openDocument(s, path); open {
			return []byte(text), nil
		}
		return os.ReadFile(path)
	}
}

// openDocument looks for a path among the documents the client has open.
//
// It walks them, which is a loop over how many files a person has open — a handful — rather
// than a map lookup, because the keys are URIs and the question is asked with a path. Making
// the state keep a second index would mean it knew what a URI means, which is the client's
// business and not its own.
func openDocument(s *state.State, path string) (string, bool) {
	want, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	for uri, text := range s.Documents() {
		open, err := filepath.Abs(textdoc.PathFromURI(lsp.URI(uri)))
		if err == nil && open == want {
			return text, true
		}
	}
	return "", false
}
