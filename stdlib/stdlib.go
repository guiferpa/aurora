// Package stdlib carries what comes with the language, so a binary can put it where it reads
// it from without anybody cloning anything.
//
// The files are read from disk at $AURORA_ROOT/lib, and that does not change: a standard
// library somebody can open, read and patch is worth more than one nobody can see. What the
// binary carries is a copy to write there, which is the difference between a toolchain that
// installs itself and one that needs a git checkout to be complete.
//
// The layout inside is the module names themselves — std/evm/storage.ar is the module
// std/evm/storage — so what is written out needs no rearranging and nothing here has to know
// what a module name looks like.
package stdlib

import (
	"embed"
	"io/fs"
)

//go:embed all:std
var files embed.FS

// Files answers what comes with the language, under the names the modules have.
func Files() fs.FS {
	return files
}
