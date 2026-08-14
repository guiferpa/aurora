package cli

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SourceExtension is what an Aurora source file is called.
const SourceExtension = ".ar"

// DefaultProfile is the profile used when no argument names one.
const DefaultProfile = "main"

// Target is what a command's positional argument resolved to.
type Target struct {
	Source   string // path of the file to compile
	Binary   string // output path from the profile; empty for a loose file
	TapeSize int    // tape width from the profile; zero means the default
	Profile  string // profile name; empty for a loose file
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
		return Target{Source: arg}, nil
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
		Source:   env.AbsPath(env.Profile.Source),
		Binary:   env.AbsPath(env.Profile.Binary),
		TapeSize: env.Profile.TapeSize,
		Profile:  name,
	}, nil
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
