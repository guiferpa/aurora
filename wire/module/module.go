// Package module is what the resolver finds and the loader consumes: which files a program
// is made of, in what order, and what is in each one.
//
// It is a wire package because those are two phases that may not know each other, and both
// have to name the same thing. Nothing here does any work.
package module

import (
	"strings"

	"github.com/guiferpa/aurora/wire/ast"
)

// An ID is a module's identity: the path it has from the source root, without the extension.
// `use a/b/c as x;` names the module a/b/c, whoever writes the line and wherever they write
// it from — which is what lets it be the key of the cache, of the environ index and of the
// prefix every name inside the module carries.
//
// The empty ID is the file somebody asked to run. It has no name because nothing imports it,
// and the names inside it are written as they were typed.
type ID string

// A Module is one file, parsed, under the name the rest of the program reaches it by.
type Module struct {
	ID   ID
	Tree ast.AST
	// Source is the text the tree was read from, which the resolver had in its hands anyway.
	// It is here for whoever has to point at a place in a file it never opened: an editor
	// jumping to where a name was declared needs the line and column of a token in another
	// file, and a position is counted in text.
	Source string
}

// IsEntry reports whether this is the file that was asked for rather than one it needed.
func (m Module) IsEntry() bool {
	return m.ID == ""
}

// Storage is the one specifier that names no file.
//
// `use storage as s;` brings in what the chain keeps between transactions, which is not a
// module of the program and has no source to read. It is written as an import because that is
// what it is to somebody using it — something reached under an alias of their choosing — and
// because giving it a keyword would spend one on it.
//
// A file may not be called this, and is refused where it is imported rather than quietly
// shadowed.
const Storage = "storage"

// Separator is what stands between a module and a name inside it: a/b/c.add.
//
// It is a character no identifier can hold — the lexer takes letters, digits and _ - ? ! > <
// — so a qualified name is never something anyone could have typed, and reading one back is
// unambiguous: exactly one of these, and neither side can contain another.
const Separator = "."

// Qualify writes a name as it is written inside a module. The parser is what calls it, when
// it decides what an identifier is called.
func Qualify(id ID, name string) string {
	if id == "" {
		return name
	}
	return string(id) + Separator + name
}

// Split reads a qualified name back: which module it belongs to, what it is called there, and
// whether it belongs to one at all. The evaluator is what calls it, to find where to look
// when a name is not in the chain.
func Split(name string) (ID, string, bool) {
	id, symbol, found := strings.Cut(name, Separator)
	if !found {
		return "", name, false
	}
	return ID(id), symbol, true
}

// Symbol is a name of this module as the file typed it, with the module taken off the front.
// It is what somebody writing `x.add` asks for, and what an error about a missing name has to
// say back to them.
func (m Module) Symbol(name string) string {
	if m.IsEntry() {
		return name
	}
	return strings.TrimPrefix(name, string(m.ID)+Separator)
}
