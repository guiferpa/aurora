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
}

// IsEntry reports whether this is the file that was asked for rather than one it needed.
func (m Module) IsEntry() bool {
	return m.ID == ""
}

// Symbol is a name of this module as the file typed it, with the module taken off the front.
// It is what somebody writing `x.add` asks for, and what an error about a missing name has to
// say back to them.
func (m Module) Symbol(name string) string {
	if m.IsEntry() {
		return name
	}
	return strings.TrimPrefix(name, string(m.ID)+".")
}
