package cli

import (
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// Every example must run. The header of each file documents the output it produces, and
// that output was pasted from a real run — this keeps them from rotting.
func TestExamplesRun(t *testing.T) {
	root := filepath.Join("..", "..", "examples")

	var sources []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".ar") {
			return nil
		}
		// A test file belongs to the source of the same name and needs it in scope, so it
		// runs under TestExamplesTestsPass rather than on its own.
		if strings.HasSuffix(d.Name(), TestExtension) {
			return nil
		}
		sources = append(sources, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walking examples: %v", err)
	}
	if len(sources) == 0 {
		t.Fatal("no examples found")
	}

	for _, source := range sources {
		t.Run(filepath.Base(source), func(t *testing.T) {
			err := Run(t.Context(), RunInput{
				Source: source,
				Stdout: io.Discard,
			})
			if err != nil {
				t.Errorf("%s failed: %v", source, err)
			}
		})
	}
}
