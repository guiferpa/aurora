package main

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/guiferpa/aurora/shared/manifest"
)

// The width of a value is the project's, and the parser needs it: it refuses a number that
// does not fit a tape, text longer than one, and a tape literal with more items than one
// holds. The server parsed everything at the default eight, so in a project written at
// sixteen the editor underlined "Guilherme" as an error while the compiler accepted it.
//
// The server has no notion of a profile — a document is a file, and the project it belongs
// to is decided by where the file is. That is the same walk the CLI does.

// projectWidth is what was found for one directory, and what says whether it still holds.
type projectWidth struct {
	manifest string // path of the manifest that answered; empty when there is no project
	modTime  time.Time
	tapeSize int
}

// stale reports whether the manifest changed or went away since it was read. The alternative
// is a server answering with the width a project had when the editor was opened.
func (w projectWidth) stale() bool {
	info, err := os.Stat(w.manifest)
	if err != nil {
		return true
	}
	return !info.ModTime().Equal(w.modTime)
}

var (
	widthsMu sync.Mutex
	// Keyed by the directory the document sits in. Only a directory that resolved to a
	// project is kept: a file with no project above it is cheap to answer for, and caching
	// the absence would outlive someone running "aurora init" next to it.
	widths = make(map[string]projectWidth)
)

// tapeSizeFor answers the width the project holding filename is written in, and zero — which
// means the default — when there is no project or its manifest cannot be read.
//
// A manifest that does not load is not the editor's to complain about: the CLI says so out
// loud, and a diagnostic about the source stays about the source.
func tapeSizeFor(filename string) int {
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		return 0
	}

	widthsMu.Lock()
	defer widthsMu.Unlock()

	if width, ok := widths[dir]; ok && !width.stale() {
		return width.tapeSize
	}
	delete(widths, dir)

	root, err := manifest.FindProjectRootFrom(dir)
	if err != nil {
		return 0
	}
	m, err := manifest.Load(root)
	if err != nil {
		return 0
	}

	path := filepath.Join(root, manifest.Filename)
	info, err := os.Stat(path)
	if err != nil {
		return m.Project.TapeSize
	}
	widths[dir] = projectWidth{manifest: path, modTime: info.ModTime(), tapeSize: m.Project.TapeSize}

	return m.Project.TapeSize
}
