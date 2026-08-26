package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/guiferpa/aurora/shared/fileutil"
)

// Putting what comes with the language where the language reads it from.
//
// The two are separate on purpose. What a program imports is read from disk, at
// $AURORA_ROOT/lib, and that is worth keeping: a standard library somebody can open and read
// is worth more than one nobody can see, and a machine that wants to patch one gets to.
//
// What this does is write it there, out of the binary, so a toolchain somebody downloaded is
// complete without a git checkout. It is the same tree either way.

// InstallStdInput is what installing takes: what to write, and where.
type InstallStdInput struct {
	// Files is the standard library as the binary carries it, under the names the modules
	// have — std/evm/storage.ar is the module std/evm/storage.
	Files fs.FS
	// Root is where the toolchain keeps its own files. Empty asks the environment.
	Root string
	// Force writes over what is already there. Without it, a directory that exists is left
	// alone and said so: somebody may have patched a module, and a command that silently
	// undoes that is a command nobody trusts twice.
	Force bool
	Out   io.Writer
}

// InstallStd writes the standard library under the toolchain's root, and says where it went.
func InstallStd(in InstallStdInput) error {
	root := in.Root
	if root == "" {
		root = fileutil.AuroraRoot()
	}
	if root == "" {
		return fmt.Errorf("there is nowhere to install to: this machine has no home directory, so set AURORA_ROOT to say where the language should keep its own files")
	}

	into := filepath.Join(root, "lib")
	if _, err := os.Stat(filepath.Join(into, "std")); err == nil && !in.Force {
		return fmt.Errorf("%s is already there, and this would write over it: pass --force to replace it, or delete it first", filepath.Join(into, "std"))
	}

	written, err := writeTree(in.Files, into)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(in.Out, "the standard library is in %s\n", filepath.Join(into, "std")); err != nil {
		return err
	}
	for _, name := range written {
		// The module names, not the paths: what somebody writes is `use std/evm/storage`, and
		// the file it reads is the compiler's business.
		if _, err := fmt.Fprintf(in.Out, "  %s\n", strings.TrimSuffix(name, ".ar")); err != nil {
			return err
		}
	}
	return nil
}

// writeTree copies a tree of files out to a directory, answering the module names it wrote.
func writeTree(files fs.FS, into string) ([]string, error) {
	written := make([]string, 0)

	err := fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || name == "." {
			return err
		}
		target := filepath.Join(into, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		source, err := fs.ReadFile(files, name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, source, 0o644); err != nil {
			return err
		}
		written = append(written, name)
		return nil
	})
	return written, err
}
