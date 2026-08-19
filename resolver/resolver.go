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
// positions and file-scoped rules are about; ID is the module the names inside belong to.
type Parse func(filename string, id module.ID, source []byte) (ast.AST, error)

// Options is what a resolver is built with.
type Options struct {
	// SourceRoot is the directory module names resolve from, and `a/b/c` under it is
	// a/b/c.ar. Empty means the caller's own directory.
	SourceRoot string
	Read       Read
	Parse      Parse
}

type Resolver struct {
	sourceRoot string
	read       Read
	parse      Parse
}

func New(opts Options) *Resolver {
	return &Resolver{sourceRoot: opts.SourceRoot, read: opts.Read, parse: opts.Parse}
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
func (r *Resolver) Resolve(entry string) ([]module.Module, error) {
	source, err := r.read(entry)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", entry, err)
	}
	tree, err := r.parse(entry, "", source)
	if err != nil {
		return nil, err
	}

	modules, err := r.Dependencies(entry, tree)
	if err != nil {
		return nil, err
	}
	return append(modules, module.Module{ID: "", Tree: tree}), nil
}

// Dependencies answers with everything the given trees need, and nothing of the trees
// themselves.
//
// It is for a caller that already has them, which is what "aurora test" is: one module
// written in two files, neither of which the resolver would have read on its own. Both are
// asked what they need, because a test names the modules it uses like any other file.
func (r *Resolver) Dependencies(entry string, trees ...ast.AST) ([]module.Module, error) {
	state := &resolution{found: make(map[module.ID]bool), entry: path.Clean(entry)}
	for _, tree := range trees {
		if err := r.dependencies(state, tree); err != nil {
			return nil, err
		}
	}
	return state.order, nil
}

// dependencies resolves every module a tree names, in the order the file names them.
//
// The declarations are top-level nodes, so this reads the list a file opens with rather than
// walking anything.
func (r *Resolver) dependencies(state *resolution, tree ast.AST) error {
	for _, node := range tree.Nodes {
		declaration, ok := node.(ast.UseDeclaration)
		if !ok {
			continue
		}
		if err := r.resolveOne(state, declaration); err != nil {
			return err
		}
	}
	return nil
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
	tree, err := r.parse(filename, id, source)
	if err != nil {
		return err
	}

	state.open = append(state.open, id)
	if err := r.dependencies(state, tree); err != nil {
		return err
	}
	state.open = state.open[:len(state.open)-1]

	// Appended after everything it needs, which is what makes the order topological.
	state.found[id] = true
	state.order = append(state.order, module.Module{ID: id, Tree: tree})
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
