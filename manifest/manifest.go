package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const Filename = "aurora.toml"

// Manifest is the parsed aurora.toml structure.
type Manifest struct {
	Project  Project             `toml:"project"`
	Profiles map[string]Profile  `toml:"profiles"`
}

// Project holds [project] section.
type Project struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

// Profile holds a profile section (e.g. [profiles.main]).
type Profile struct {
	Entrypoint      string `toml:"entrypoint"`
	Target          string `toml:"target"`
	RPCURL          string `toml:"rpc_url"`
	PrivateKeyPath  string `toml:"private_key_path"`
	ContractAddress string `toml:"contract_address"`
}

// FindProjectRoot returns the directory that contains aurora.toml, starting from the current directory and walking up. Returns an error if not found.
func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		path := filepath.Join(dir, Filename)
		if _, err := os.Stat(path); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%s not found in current directory or any parent (run 'aurora init' to create a project manifest)", Filename)
		}
		dir = parent
	}
}

// Load reads and parses the manifest from the given project root directory.
func Load(projectRoot string) (*Manifest, error) {
	path := filepath.Join(projectRoot, Filename)
	var m Manifest
	if _, err := toml.DecodeFile(path, &m); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if m.Profiles == nil {
		m.Profiles = make(map[string]Profile)
	}
	return &m, nil
}

// Profile returns the named profile (e.g. "main") or an error if missing.
func (m *Manifest) Profile(name string) (Profile, error) {
	p, ok := m.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("profile %q not found in manifest", name)
	}
	return p, nil
}

// AbsPath returns path joined with project root (for entrypoint, target, private_key_path).
func AbsPath(projectRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(projectRoot, path)
}
