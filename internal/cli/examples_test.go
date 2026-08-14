package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every example in examples/ must run. The header of each file documents the output it
// produces, and that output was pasted from a real run — this keeps them from rotting.
func TestExamplesRun(t *testing.T) {
	dir := filepath.Join("..", "..", "examples")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading examples: %v", err)
	}

	var found int
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".ar") {
			continue
		}
		found++
		t.Run(name, func(t *testing.T) {
			err := Run(t.Context(), RunInput{
				Source: filepath.Join(dir, name),
				Stdout: io.Discard,
			})
			if err != nil {
				t.Errorf("example failed: %v", err)
			}
		})
	}

	if found == 0 {
		t.Fatal("no examples found")
	}
}
