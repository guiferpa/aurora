package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/guiferpa/aurora/manifest"
)

// InitManifestTemplate is the template for aurora.toml. Use fmt.Sprintf with projectName for the name field.
const InitManifestTemplate = `# Aurora project manifest.
# See https://github.com/guiferpa/aurora for more information.

[project]
# Project identifier (inherited from the root folder name where 'aurora init' was run).
name = %q
# Project version (semantic version recommended).
version = "0.1.0"

[profiles.main]
# Default profile. Commands like 'aurora build' or 'aurora run' use these paths when no file is given.
# Path to the main source file (entrypoint). Used by build, run, and deploy when no file argument is passed.
entrypoint = "src/main.ar"
# Path where the compiled binary is written. Name matches the entrypoint filename (without extension). Used by 'aurora build' when no -o output is passed.
target = "dist/main"
`

// InitInput is the input for the Init handler.
type InitInput struct {
	Dir         string // directory to write aurora.toml (usually cwd)
	ProjectName string // default: filepath.Base(Dir)
}

// Init creates aurora.toml in Dir with project name from InitInput.ProjectName (or from Dir base if empty).
func Init(in InitInput) error {
	if in.Dir == "" {
		return fmt.Errorf("InitInput.Dir is required")
	}
	path := filepath.Join(in.Dir, manifest.Filename)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", manifest.Filename)
	}
	name := in.ProjectName
	if name == "" {
		name = filepath.Base(in.Dir)
	}
	content := fmt.Sprintf(InitManifestTemplate, name)
	return os.WriteFile(path, []byte(content), 0o644)
}
