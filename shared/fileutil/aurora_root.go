package fileutil

import (
	"os"
	"path/filepath"
)

// Where the toolchain keeps what comes with the language.
//
// It is one directory and everything under it belongs to the version of the toolchain that
// installed it, which is the whole of the versioning story: there is no way for a program to
// ask for another version, the way there is none in C. What answers `use std/evm/storage` is
// what the aurora on this machine shipped with.

// AuroraRootVariable is what somebody sets to keep it somewhere else.
const AuroraRootVariable = "AURORA_ROOT"

// AuroraRoot answers where the toolchain's own files live: what AURORA_ROOT says, or
// $HOME/.aurora when it says nothing.
//
// A machine with neither answers empty rather than guessing, and a program that imports a
// module of the language is refused where it wrote the import — which says more than a path
// that was never going to exist.
func AuroraRoot() string {
	if named := os.Getenv(AuroraRootVariable); named != "" {
		return named
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".aurora")
}

// StdRoot answers the directory the language's own modules read from, which is lib under the
// root: `std/evm/storage` is $AURORA_ROOT/lib/std/evm/storage.ar.
//
// The prefix stays in the path rather than being stripped, because it is part of the module's
// name — two files at lib/std/evm/storage.ar and lib/evm/storage.ar would be two modules, and
// only one of them is importable.
func StdRoot() string {
	root := AuroraRoot()
	if root == "" {
		return ""
	}
	return filepath.Join(root, "lib")
}
