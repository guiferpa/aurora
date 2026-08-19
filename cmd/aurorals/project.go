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

// projectSettings is what was found for one directory, and what says whether it still holds.
//
// Two things come out of the same walk — how wide a value is, and where a module name
// resolves from — so they are read together and kept together.
type projectSettings struct {
	manifest string // path of the manifest that answered; empty when there is no project
	modTime  time.Time
	tapeSize int
	// sourceRoot is absolute. A server has no working directory worth speaking of, so where
	// a module name resolves from is the project's root joined with what the manifest says
	// — which is what the command line arrives at too, whenever it is run from that root.
	sourceRoot string
}

// stale reports whether the manifest changed or went away since it was read. The alternative
// is a server answering with the width a project had when the editor was opened.
func (w projectSettings) stale() bool {
	info, err := os.Stat(w.manifest)
	if err != nil {
		return true
	}
	return !info.ModTime().Equal(w.modTime)
}

var (
	projectsMu sync.Mutex
	// Keyed by the directory the document sits in. Only a directory that resolved to a
	// project is kept: a file with no project above it is cheap to answer for, and caching
	// the absence would outlive someone running "aurora init" next to it.
	projects = make(map[string]projectSettings)
)

// settingsFor answers what the project holding filename says, and whether there is one.
//
// A manifest that does not load is not the editor's to complain about: the CLI says so out
// loud, and a diagnostic about the source stays about the source.
func settingsFor(filename string) (projectSettings, bool) {
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		return projectSettings{}, false
	}

	projectsMu.Lock()
	defer projectsMu.Unlock()

	if found, ok := projects[dir]; ok && !found.stale() {
		return found, true
	}
	delete(projects, dir)

	root, err := manifest.FindProjectRootFrom(dir)
	if err != nil {
		return projectSettings{}, false
	}
	m, err := manifest.Load(root)
	if err != nil {
		return projectSettings{}, false
	}

	found := projectSettings{
		tapeSize:   m.Project.TapeSize,
		sourceRoot: filepath.Join(root, m.SourceRoot()),
	}

	path := filepath.Join(root, manifest.Filename)
	info, err := os.Stat(path)
	if err != nil {
		return found, true
	}
	found.manifest = path
	found.modTime = info.ModTime()
	projects[dir] = found

	return found, true
}

// tapeSizeFor answers the width the project holding filename is written in, and zero — which
// means the default — when there is no project.
func tapeSizeFor(filename string) int {
	if found, ok := settingsFor(filename); ok {
		return found.tapeSize
	}
	return 0
}

// sourceRootFor answers where a module name resolves from for this document: what the project
// says, and src/ next to the file when it belongs to no project.
func sourceRootFor(filename string) string {
	if found, ok := settingsFor(filename); ok {
		return found.sourceRoot
	}
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		return manifest.DefaultSourceRoot
	}
	return filepath.Join(dir, manifest.DefaultSourceRoot)
}
