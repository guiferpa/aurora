package manifest

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPersistDeploy_createsDeploysFile(t *testing.T) {
	dir := t.TempDir()
	auroraPath := filepath.Join(dir, Filename)
	auroraContent := `# My project
[project]
name = "test"
version = "0.1.0"

[profiles.main]
source = "src/main.ar"
binary = "bin/main"
`
	if err := os.WriteFile(auroraPath, []byte(auroraContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := PersistDeploy(dir, "main", "0xabc", "0xtxhash", "2025-02-05T12:00:00Z"); err != nil {
		t.Fatal(err)
	}
	// aurora.toml must be unchanged
	written, _ := os.ReadFile(auroraPath)
	if !bytes.Equal(bytes.TrimSpace(written), bytes.TrimSpace([]byte(auroraContent))) {
		t.Error("aurora.toml must not be modified by PersistDeploy")
	}
	// .aurora.deploys.toml must exist with header and deploy state
	deploysPath := filepath.Join(dir, DeploysFilename)
	content, err := os.ReadFile(deploysPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("Do not edit")) {
		t.Error("deploys file should contain header comment")
	}
	if !bytes.Contains(content, []byte("[deploys.main]")) {
		t.Error("deploys file should contain deploy section for profile")
	}
	if !bytes.Contains(content, []byte("0xabc")) {
		t.Error("deploys file should contain contract_address")
	}
}

func TestPersistDeploy_preservesOtherProfiles(t *testing.T) {
	dir := t.TempDir()
	writeAuroraToml(t, dir)
	// Deploy main
	if err := PersistDeploy(dir, "main", "0xmain", "0xtxmain", "2025-02-05T12:00:00Z"); err != nil {
		t.Fatal(err)
	}
	// Deploy sepolia (simulate: we only have PersistDeploy, so we need to write deploys file with main then add sepolia by loading and re-persisting)
	// Actually: load the file, add sepolia manually to state, then call PersistDeploy for sepolia - that will load existing state (main) and add sepolia.
	if err := PersistDeploy(dir, "sepolia", "0xsepolia", "0xtxsepolia", "2025-02-05T13:00:00Z"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, DeploysFilename))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("0xmain")) || !bytes.Contains(content, []byte("0xsepolia")) {
		t.Error("deploys file should contain both main and sepolia after two deploys")
	}
	// Overwrite main again; sepolia must still be there
	if err := PersistDeploy(dir, "main", "0xmain2", "0xtxmain2", "2025-02-05T14:00:00Z"); err != nil {
		t.Fatal(err)
	}
	content, _ = os.ReadFile(filepath.Join(dir, DeploysFilename))
	if !bytes.Contains(content, []byte("0xsepolia")) {
		t.Error("sepolia deploy state must be preserved when updating main")
	}
	if !bytes.Contains(content, []byte("0xmain2")) {
		t.Error("main deploy state must be updated")
	}
}

func TestLoad_readsDeployStateFromDeploysFile(t *testing.T) {
	dir := t.TempDir()
	writeAuroraToml(t, dir)
	if err := PersistDeploy(dir, "main", "0xloaded", "0xtx", "2025-02-05T12:00:00Z"); err != nil {
		t.Fatal(err)
	}
	m, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Deploys["main"].ContractAddress != "0xloaded" {
		t.Errorf("Load should read deploy state from .aurora.deploys.toml, got contract_address %q", m.Deploys["main"].ContractAddress)
	}
}

func TestLoad_deploysEmptyWhenNoDeploysFile(t *testing.T) {
	dir := t.TempDir()
	writeAuroraToml(t, dir)
	m, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Deploys == nil {
		t.Fatal("Deploys map should be non-nil")
	}
	if len(m.Deploys) != 0 {
		t.Errorf("Deploys should be empty when .aurora.deploys.toml is missing, got %d entries", len(m.Deploys))
	}
}

func writeAuroraToml(t *testing.T, dir string) {
	t.Helper()
	content := `[project]
name = "test"
version = "0.1.0"

[profiles.main]
source = "src/main.ar"
binary = "bin/main"
`
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
