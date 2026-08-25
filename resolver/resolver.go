// Package resolver answers which files a program is made of, and in what order they load.
//
// It starts at the file somebody asked to run, reads what that file says it depends on, and
// keeps going until nothing is left to find. What comes back is every module, dependencies
// before dependents, each parsed once no matter how many files name it.
//
// It touches nothing. Source arrives through a port and trees come back from another, so the
// same resolver serves a command line reading a disk and a page in a browser holding its
// files in memory.
package resolver

import (
	"fmt"
	"path"
	"strings"

	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
	"github.com/guiferpa/aurora/wire/token"
)

// Extension is what a module's file is called. A specifier leaves it out — `use a/b/c as x;`
// reads a/b/c.ar — because it names a module and not a path.
const Extension = ".ar"

// Read hands back the source of a file. It is how the world arrives here: the command line
// reads a disk, the playground reads a map it already has, a test reads whatever it likes.
type Read func(path string) ([]byte, error)

// Parse turns one module's source into a tree. Filename is where it came from, which is what
// positions and file-scoped rules are about; ID is the module the names inside belong to; and
// imports is what the modules it names promised, which it needs while parsing because a shape
// is resolved there and a shape's name never leaves the file that declared it.
type Parse func(filename string, id module.ID, source []byte, imports map[string]ast.Offer) (ast.AST, error)

// Header answers what a source imports, without parsing it.
//
// It exists because a module has to be read before whoever imports it — the parse of a file
// needs what its dependencies promised — and knowing what to read first cannot itself require
// a parse. The declarations are the top of the file, so reading the top is enough.
type Header func(source []byte) ([]ast.UseDeclaration, error)

// Options is what a resolver is built with.
type Options struct {
	// SourceRoot is the directory module names resolve from, and `a/b/c` under it is
	// a/b/c.ar. Empty means the caller's own directory.
	SourceRoot string
	Read       Read
	Parse      Parse
	Header     Header
}

type Resolver struct {
	sourceRoot string
	read       Read
	parse      Parse
	header     Header
}

func New(opts Options) *Resolver {
	return &Resolver{sourceRoot: opts.SourceRoot, read: opts.Read, parse: opts.Parse, header: opts.Header}
}

// resolution is one call to Resolve: what has been found, what is being looked at right now,
// and the order the answer is being built in.
type resolution struct {
	found map[module.ID]bool
	order []module.Module
	// open is the chain of modules being resolved, innermost last. A specifier that is
	// already on it is a cycle, and the chain is the error.
	open []module.ID
	// entry is the path of the file that was asked for, so a module cannot import it: it
	// would be read twice, under two names, and its top level would run twice.
	entry string
}

// Resolve answers with the entry and everything it needs, dependencies first.
//
// The entry is a path rather than a module name because it is the file somebody asked to
// run, and it is the one module with no id.
// Resolve answers with the entry and everything it needs, dependencies first.
//
// The entry is parsed last, which is the whole shape of this: a file is read before the file
// that imports it, so by the time one is parsed, everything it named has been. Knowing what to
// read first is what the header is for.
func (r *Resolver) Resolve(entry string) ([]module.Module, error) {
	source, err := r.read(entry)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", entry, err)
	}
	uses, err := r.header(source)
	if err != nil {
		return nil, err
	}

	modules, err := r.DependenciesOf(entry, uses)
	if err != nil {
		return nil, err
	}

	tree, err := r.parse(entry, "", source, OffersOf(modules))
	if err != nil {
		return nil, err
	}
	return append(modules, module.Module{ID: "", Tree: tree, Source: string(source)}), nil
}

// OffersOf is what a set of modules offer, by the name they are imported under.
//
// Everything found so far is handed over rather than only what one file names: a parse looks
// up the specifiers it actually wrote, so an entry nobody asks for costs nothing.
func OffersOf(modules []module.Module) map[string]ast.Offer {
	offers := make(map[string]ast.Offer, len(modules))
	for _, each := range modules {
		if len(each.Tree.Returns) == 0 && len(each.Tree.Shapes) == 0 {
			continue
		}
		offers[string(each.ID)] = ast.Offer{Shapes: each.Tree.Shapes, Returns: each.Tree.Returns}
	}
	return offers
}

// Dependencies answers with everything the given trees need, and nothing of the trees
// themselves.
//
// It is for a caller that already has them, which is what "aurora test" is: one module
// written in two files, neither of which the resolver would have read on its own. Both are
// asked what they need, because a test names the modules it uses like any other file.
func (r *Resolver) Dependencies(entry string, trees ...ast.AST) ([]module.Module, error) {
	uses := make([]ast.UseDeclaration, 0)
	for _, tree := range trees {
		uses = append(uses, declarationsOf(tree)...)
	}
	return r.DependenciesOf(entry, uses)
}

// DependenciesOf answers the same, for a caller that has the declarations without a tree.
//
// An editor is that caller. A document is too broken to parse most of the time somebody is
// looking at it — the moment a dot is typed there is no name after it yet — and what is
// inside a module is exactly what is wanted then. The declarations are the top of the file
// and readable from the tokens alone, so they arrive that way.
func (r *Resolver) DependenciesOf(entry string, uses []ast.UseDeclaration) ([]module.Module, error) {
	state := &resolution{found: make(map[module.ID]bool), entry: path.Clean(entry)}
	for _, declaration := range uses {
		if err := r.resolveOne(state, declaration); err != nil {
			return nil, err
		}
	}
	return state.order, nil
}

// declarationsOf reads the imports out of a tree. They are top-level nodes, so this reads the
// list a file opens with rather than walking anything.
func declarationsOf(tree ast.AST) []ast.UseDeclaration {
	uses := make([]ast.UseDeclaration, 0)
	for _, node := range tree.Nodes {
		if declaration, ok := node.(ast.UseDeclaration); ok {
			uses = append(uses, declaration)
		}
	}
	return uses
}

// resolveOne finds one module, everything under it, and adds it to the order.
//
// A module already found is not read again: several files naming the same one is the ordinary
// case, and it loads once — its body runs once, which is the whole premise of a module being
// a file that executes.
func (r *Resolver) resolveOne(state *resolution, declaration ast.UseDeclaration) error {
	id := module.ID(declaration.Specifier)
	if state.found[id] {
		return nil
	}
	if err := state.refuseCycle(id, declaration); err != nil {
		return err
	}

	filename := r.Filename(id)
	if path.Clean(filename) == state.entry {
		return token.NewError(declaration.Token, "%s is the file being run at line %d and column %d: a program cannot import itself",
			id, declaration.Token.GetLine(), declaration.Token.GetColumn())
	}

	source, err := r.read(filename)
	if err != nil {
		return token.NewError(declaration.Token, "module %s is not there at line %d and column %d: no %s",
			id, declaration.Token.GetLine(), declaration.Token.GetColumn(), filename)
	}
	uses, err := r.header(source)
	if err != nil {
		return err
	}

	// Everything this module names is read before it is, which is what lets its parse be
	// handed what they promised.
	state.open = append(state.open, id)
	for _, use := range uses {
		if err := r.resolveOne(state, use); err != nil {
			return err
		}
	}
	state.open = state.open[:len(state.open)-1]

	tree, err := r.parse(filename, id, source, OffersOf(state.order))
	if err != nil {
		// Which file it was. The entry is the file somebody named and needs no introduction;
		// a module is one they may not have opened, and a line and column alone name nothing.
		return fmt.Errorf("%s: %w", filename, err)
	}

	// Appended after everything it needs, which is what makes the order topological.
	state.found[id] = true
	state.order = append(state.order, module.Module{ID: id, Tree: tree, Source: string(source)})
	return nil
}

// Filename is the file a module name reads: a/b/c under the source root, with the extension
// back on.
func (r *Resolver) Filename(id module.ID) string {
	return path.Join(r.sourceRoot, string(id)+Extension)
}

// refuseCycle answers with the whole chain rather than the two ends of it, because a cycle of
// four is read by following it and the middle is where the mistake usually is.
func (s *resolution) refuseCycle(id module.ID, declaration ast.UseDeclaration) error {
	for i, open := range s.open {
		if open != id {
			continue
		}
		chain := make([]string, 0, len(s.open)-i+1)
		for _, each := range s.open[i:] {
			chain = append(chain, string(each))
		}
		chain = append(chain, string(id))
		return token.NewError(declaration.Token, "modules go in a circle at line %d and column %d: %s",
			declaration.Token.GetLine(), declaration.Token.GetColumn(), strings.Join(chain, " → "))
	}
	return nil
}
