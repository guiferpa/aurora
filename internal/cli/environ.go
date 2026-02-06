package cli

import (
	"github.com/guiferpa/aurora/manifest"
)

// Environ holds project root, loaded manifest, and the selected profile.
// Commands that need the manifest use LoadEnviron once and then AbsPath, Profile, etc.
type Environ struct {
	Root    string
	Manifest *manifest.Manifest
	Profile manifest.Profile
}

// LoadEnviron finds project root, loads the manifest, and returns Environ for the given profile.
func LoadEnviron(profileName string) (*Environ, error) {
	root, err := manifest.FindProjectRoot()
	if err != nil {
		return nil, err
	}
	m, err := manifest.Load(root)
	if err != nil {
		return nil, err
	}
	prof, err := m.Profile(profileName)
	if err != nil {
		return nil, err
	}
	return &Environ{Root: root, Manifest: m, Profile: prof}, nil
}

// AbsPath returns path joined with the project root (for source, binary, privkey).
func (e *Environ) AbsPath(path string) string {
	return manifest.AbsPath(e.Root, path)
}

// RequireManifest ensures aurora.toml exists in the current directory or any parent.
// It can be used by the root command's PersistentPreRunE for commands that need a project.
func RequireManifest() error {
	_, err := manifest.FindProjectRoot()
	return err
}
