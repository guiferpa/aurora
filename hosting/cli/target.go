package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/guiferpa/aurora/shared/manifest"
)

// SourceExtension is what an Aurora source file is called.
const SourceExtension = ".ar"

// DefaultProfile is the profile used when no argument names one.
const DefaultProfile = "main"

// Target is what a command's positional argument resolved to.
type Target struct {
	Source   string // path of the file to compile
	Binary   string // output path from the profile; empty for a loose file
	TapeSize int    // tape width of the project the file belongs to; zero means the default
	Profile  string // profile name; empty for a loose file
	// SourceRoot is where module names resolve from, relative to the directory the command
	// was run in — with a manifest or without, which is what keeps the rule one sentence.
	SourceRoot string
}

// FromProfile reports whether the target came from the manifest rather than a path.
func (t Target) FromProfile() bool {
	return t.Profile != ""
}

// ResolveTarget turns one positional argument into what to compile.
//
//	(none)      the "main" profile
//	dev         the "dev" profile
//	path/x.ar   that file, with no manifest involved
//
// The extension decides: a profile name never ends in .ar and an Aurora file always does.
// Running a loose file is the common case while learning the language, and it should not
// require a project to exist — so this path never looks for aurora.toml.
func ResolveTarget(arg string) (Target, error) {
	if strings.HasSuffix(arg, SourceExtension) {
		tapeSize, err := ProjectTapeSize(arg)
		if err != nil {
			return Target{}, err
		}
		return Target{Source: arg, TapeSize: tapeSize, SourceRoot: ProjectSourceRoot(arg)}, nil
	}

	if arg != "" && looksLikePath(arg) {
		return Target{}, fmt.Errorf("%q is neither a profile name nor an %s file", arg, SourceExtension)
	}

	name := arg
	if name == "" {
		name = DefaultProfile
	}

	env, err := LoadEnviron(name)
	if err != nil {
		return Target{}, err
	}

	return Target{
		Source:     env.AbsPath(env.Profile.Source),
		Binary:     env.AbsPath(env.Profile.Binary),
		TapeSize:   env.Manifest.Project.TapeSize,
		Profile:    name,
		SourceRoot: env.Manifest.SourceRoot(),
	}, nil
}

// ProjectTapeSize answers the width of the project a file sits in, and zero when it sits in
// none.
//
// A loose file does not require a project to exist — that is what keeps trying the language
// out cheap — but when it is inside one it is written in that project's dialect, the same as
// the file a profile names. Reading the same file two widths depending on how it was named
// is the bug this closes.
func ProjectTapeSize(source string) (int, error) {
	dir, err := filepath.Abs(filepath.Dir(source))
	if err != nil {
		return 0, err
	}
	root, err := manifest.FindProjectRootFrom(dir)
	if err != nil {
		return 0, nil // no project: the default width applies
	}
	m, err := manifest.Load(root)
	if err != nil {
		return 0, err
	}
	return m.Project.TapeSize, nil
}

// ProjectSourceRoot answers where module names resolve from for a file named by path, which
// is what the project it sits in says, or "src" when it sits in none.
//
// The root is relative to the directory the command was run in either way. That is one rule
// with no exception, and the price of it is that a project with modules is run from its own
// root — which is what the error says when a module is not found there.
func ProjectSourceRoot(source string) string {
	dir, err := filepath.Abs(filepath.Dir(source))
	if err != nil {
		return manifest.DefaultSourceRoot
	}
	root, err := manifest.FindProjectRootFrom(dir)
	if err != nil {
		return manifest.DefaultSourceRoot
	}
	m, err := manifest.Load(root)
	if err != nil {
		return manifest.DefaultSourceRoot
	}
	return m.SourceRoot()
}

// looksLikePath catches an argument that was meant as a file but lost its extension, so
// the error names the real problem instead of reporting a missing profile.
func looksLikePath(arg string) bool {
	return strings.ContainsRune(arg, '/') || strings.ContainsRune(arg, filepath.Separator) ||
		strings.HasPrefix(arg, ".") || filepath.Ext(arg) != ""
}

// DefaultBinaryPath is where "aurora build" writes when a loose file is compiled and no
// output was given: the file's name without its extension, in the working directory.
// A loose file has no profile, so there is no binary path to inherit.
func DefaultBinaryPath(source string) string {
	base := filepath.Base(source)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
