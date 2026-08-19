// Package loader turns the modules a resolver found into a program.
//
// Its first job is the one the last attempt at modules never had: checking that a qualified
// name is really there. The parser writes `x.add` down as a/b/c.add because the use above it
// says so, and it cannot know whether a/b/c has an add — only whoever holds every module
// does, which is here.
package loader

import (
	"slices"
	"strings"

	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
	"github.com/guiferpa/aurora/wire/token"
)

// Exports is what a module offers whoever imports it: the names it binds with ident at the
// top of the file, in the order it binds them, as they were typed.
//
// A struct is not among them, because a struct does not cross a module. Neither is anything
// bound inside a block or a deferred body: that lives in an environ which does not exist
// until the body runs. A defer needs no special case at all — its value is its index, as a
// tape, so it is already what an ident binds.
func Exports(m module.Module) []string {
	names := make([]string, 0, len(m.Tree.Nodes))
	for _, node := range m.Tree.Nodes {
		binding, ok := node.(ast.IdentLiteral)
		if !ok {
			continue
		}
		names = append(names, m.Symbol(binding.Id))
	}
	return names
}

// Check answers whether every qualified name in every module is really there.
//
// It reads the whole set rather than one module at a time because a reference points sideways:
// the file that wrote it and the file that has to have the name are two different ones.
func Check(modules []module.Module) error {
	offered := make(map[module.ID]map[string]bool, len(modules))
	for _, each := range modules {
		names := make(map[string]bool)
		for _, name := range Exports(each) {
			names[name] = true
		}
		offered[each.ID] = names
	}

	for _, each := range modules {
		for _, reference := range each.Tree.References {
			if err := check(reference, offered); err != nil {
				return err
			}
		}
	}
	return nil
}

// check reads one qualified name against what its module offers.
func check(reference ast.Reference, offered map[module.ID]map[string]bool) error {
	id := module.ID(reference.Module)
	names, loaded := offered[id]
	if !loaded {
		return token.NewError(reference.Token, "module %s was never loaded at line %d and column %d",
			reference.Module, reference.Token.GetLine(), reference.Token.GetColumn())
	}
	if names[reference.Symbol] {
		return nil
	}
	return token.NewError(reference.Token, "module %s has no %s at line %d and column %d (it has %s)",
		reference.Module, reference.Symbol, reference.Token.GetLine(), reference.Token.GetColumn(), listing(names))
}

// listing names what a module does offer, so the message is enough to fix the line with. The
// order is the map's, so it is sorted: a message that changed between two runs of the same
// compiler would be worse than one that reads out of order.
func listing(names map[string]bool) string {
	if len(names) == 0 {
		return "nothing"
	}
	offered := make([]string, 0, len(names))
	for name := range names {
		offered = append(offered, name)
	}
	slices.Sort(offered)
	return strings.Join(offered, ", ")
}
