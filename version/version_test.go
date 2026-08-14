package version

import "testing"

// VERSION is filled in at build time via ldflags; a local build keeps the default. It is
// reported by "aurora version" and by the language server on initialize, so it must never
// be empty.
func TestVersionIsSet(t *testing.T) {
	if VERSION == "" {
		t.Error("VERSION is empty")
	}
}
