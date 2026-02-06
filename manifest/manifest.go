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
	Project  Project                `toml:"project"`
	Profiles map[string]Profile     `toml:"profiles"`
	Deploys  map[string]DeployState `toml:"deploys"`
}

// DeployState holds the last deploy result for a profile. Written by the CLI on each deploy; do not edit by hand.
type DeployState struct {
	ContractAddress string `toml:"contract_address"`
	DeployedAt      string `toml:"deployed_at"` // RFC3339
}

// Project holds [project] section.
type Project struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

// Profile holds a profile section (e.g. [profiles.main]).
type Profile struct {
	Source  string `toml:"source"`
	Binary  string `toml:"binary"`
	RPC     string `toml:"rpc"`
	Privkey string `toml:"privkey"`
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
	if m.Deploys == nil {
		m.Deploys = make(map[string]DeployState)
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

// AbsPath returns path joined with project root (for source, binary, privkey).
func AbsPath(projectRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(projectRoot, path)
}

// PersistDeploy updates the manifest with the latest deploy for the given profile. Overwrites [deploys.<profileName>] with address and deployedAt (RFC3339). Call after a successful deploy.
func PersistDeploy(projectRoot, profileName, address, deployedAt string) error {
	m, err := Load(projectRoot)
	if err != nil {
		return err
	}
	if m.Deploys == nil {
		m.Deploys = make(map[string]DeployState)
	}
	m.Deploys[profileName] = DeployState{ContractAddress: address, DeployedAt: deployedAt}
	path := filepath.Join(projectRoot, Filename)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("open manifest for write: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close manifest file: %w", cerr)
		}
	}()
	if err := toml.NewEncoder(f).Encode(m); err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	return nil
}
