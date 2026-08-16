package textdoc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// project writes a manifest and a source file, and answers with the path of the source.
func project(t *testing.T, manifest string) string {
	t.Helper()
	dir := t.TempDir()
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(dir, "aurora.toml"), []byte(manifest), 0o644); err != nil {
			t.Fatalf("writing the manifest: %v", err)
		}
	}
	return filepath.Join(dir, "main.ar")
}

// "Guilherme" is nine bytes: it does not fit the default tape and does fit a sixteen-byte
// one. The server read every document at the default, so a project written at sixteen had
// this underlined in the editor while the compiler took it without a word.
const nineBytes = `printc "Guilherme";`

func TestADocumentIsReadInTheWidthOfItsProject(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		wantErr  bool
	}{
		{
			name:     "a project wide enough",
			manifest: "[project]\nname = \"p\"\ntape_size = 16\n",
		},
		{
			name:     "a project that says nothing keeps the default",
			manifest: "[project]\nname = \"p\"\n",
			wantErr:  true,
		},
		{
			name:    "a file with no project keeps the default",
			wantErr: true,
		},
		{
			name:     "a project narrower than the default",
			manifest: "[project]\nname = \"p\"\ntape_size = 1\n",
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := project(t, tc.manifest)

			diagnostics := ValidateCode(source, nineBytes)

			if tc.wantErr && len(diagnostics) == 0 {
				t.Error("the text was accepted, want it reported as too long for a tape")
			}
			if !tc.wantErr && len(diagnostics) != 0 {
				t.Errorf("the text was reported: %v", diagnostics[0].Message)
			}
		})
	}
}

// A number too large for the project's tape is the other half of the same rule.
func TestANumberIsCheckedAgainstTheProjectsTape(t *testing.T) {
	source := project(t, "[project]\nname = \"p\"\ntape_size = 1\n")

	diagnostics := ValidateCode(source, "printd 300;")

	if len(diagnostics) == 0 {
		t.Fatal("300 was accepted on a one-byte tape")
	}
	if !strings.Contains(diagnostics[0].Message, "1-byte tape") {
		t.Errorf("the diagnostic says %q, want it to name the project's width", diagnostics[0].Message)
	}
}

// A manifest edited while the editor is open has to reach the next keystroke: the answer is
// cached per directory, and the cache is only good while the manifest is the one that was
// read.
func TestTheWidthFollowsTheManifestBeingEdited(t *testing.T) {
	source := project(t, "[project]\nname = \"p\"\ntape_size = 1\n")
	path := filepath.Join(filepath.Dir(source), "aurora.toml")

	if len(ValidateCode(source, nineBytes)) == 0 {
		t.Fatal("the text was accepted on a one-byte tape")
	}

	if err := os.WriteFile(path, []byte("[project]\nname = \"p\"\ntape_size = 16\n"), 0o644); err != nil {
		t.Fatalf("rewriting the manifest: %v", err)
	}
	// Modification times have a resolution of their own, and a rewrite inside the same tick
	// would read as unchanged.
	now := timeAfter(t, path)
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatalf("touching the manifest: %v", err)
	}

	if diagnostics := ValidateCode(source, nineBytes); len(diagnostics) != 0 {
		t.Errorf("the widened project still reports %q", diagnostics[0].Message)
	}
}

// timeAfter answers with a modification time later than the file's, so a rewrite is visible
// however coarse the filesystem's clock is.
func timeAfter(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("reading the manifest: %v", err)
	}
	return info.ModTime().Add(time.Second)
}

// A manifest that does not load leaves the document at the default rather than turning a
// project's mistake into a diagnostic about the source.
func TestADocumentSurvivesAManifestItCannotRead(t *testing.T) {
	source := project(t, "[project\nname = ")

	if diagnostics := ValidateCode(source, "printd 1;"); len(diagnostics) != 0 {
		t.Errorf("a broken manifest was reported as a problem with the source: %v", diagnostics[0].Message)
	}
}
