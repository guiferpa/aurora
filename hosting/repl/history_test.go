package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadHistoryMissingFileIsEmpty(t *testing.T) {
	h := LoadHistory(filepath.Join(t.TempDir(), "does", "not", "exist"))
	if h.Len() != 0 {
		t.Errorf("expected empty history, got %d entries", h.Len())
	}
	if got := h.At(0); got != "" {
		t.Errorf("expected empty entry, got %q", got)
	}
}

func TestAppendPersistsBetweenSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".aurora", "history")

	first := LoadHistory(path)
	for _, line := range []string{"ident a = 1;", "a + 1;"} {
		if err := first.Append(line); err != nil {
			t.Fatalf("append %q: %v", line, err)
		}
	}

	second := LoadHistory(path)
	if second.Len() != 2 {
		t.Fatalf("expected 2 entries after reload, got %d", second.Len())
	}
	if got, want := second.At(0), "ident a = 1;"; got != want {
		t.Errorf("entry 0 = %q, want %q", got, want)
	}
	if got, want := second.At(1), "a + 1;"; got != want {
		t.Errorf("entry 1 = %q, want %q", got, want)
	}
}

func TestAppendCreatesPrivateFileAndDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".aurora")
	path := filepath.Join(dir, "history")

	if err := LoadHistory(path).Append("ident a = 1;"); err != nil {
		t.Fatalf("append: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 600", perm)
	}

	di, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir perm = %o, want 700", perm)
	}
}

func TestAppendSkipsEmptyAndConsecutiveDuplicates(t *testing.T) {
	h := LoadHistory(filepath.Join(t.TempDir(), "history"))

	for _, line := range []string{"a;", "a;", "", "   ", "b;", "a;"} {
		if err := h.Append(line); err != nil {
			t.Fatalf("append %q: %v", line, err)
		}
	}

	want := []string{"a;", "b;", "a;"}
	if h.Len() != len(want) {
		t.Fatalf("expected %d entries, got %d", len(want), h.Len())
	}
	for i, w := range want {
		if got := h.At(i); got != w {
			t.Errorf("entry %d = %q, want %q", i, got, w)
		}
	}
}

func TestLoadHistoryCompactsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")

	var content strings.Builder
	total := historyRewriteAt + 10
	for i := 0; i < total; i++ {
		fmt.Fprintf(&content, "line %d;\n", i)
	}
	if err := os.WriteFile(path, []byte(content.String()), 0o600); err != nil {
		t.Fatalf("write history: %v", err)
	}

	h := LoadHistory(path)
	if h.Len() != HistoryLimit {
		t.Fatalf("expected %d entries in memory, got %d", HistoryLimit, h.Len())
	}
	if got, want := h.At(h.Len()-1), fmt.Sprintf("line %d;", total-1); got != want {
		t.Errorf("newest entry = %q, want %q", got, want)
	}

	bs, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read compacted file: %v", err)
	}
	if lines := strings.Count(string(bs), "\n"); lines != HistoryLimit {
		t.Errorf("compacted file has %d lines, want %d", lines, HistoryLimit)
	}
}

func TestMemoryOnlyHistoryWritesNoFile(t *testing.T) {
	h := LoadHistory("")
	if err := h.Append("a;"); err != nil {
		t.Fatalf("append: %v", err)
	}
	if h.Len() != 1 {
		t.Errorf("expected 1 entry in memory, got %d", h.Len())
	}
}

func TestNilHistoryIsUsable(t *testing.T) {
	var h *History
	if err := h.Append("a;"); err != nil {
		t.Errorf("append on nil history: %v", err)
	}
	if h.Len() != 0 || h.At(0) != "" {
		t.Errorf("nil history should look empty")
	}
}

func TestDefaultHistoryPath(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "custom-history")
	t.Setenv(HistoryEnv, custom)
	got, err := DefaultHistoryPath()
	if err != nil {
		t.Fatalf("with %s set: %v", HistoryEnv, err)
	}
	if got != custom {
		t.Errorf("path = %q, want %q", got, custom)
	}

	t.Setenv(HistoryEnv, "")
	got, err = DefaultHistoryPath()
	if err != nil {
		t.Fatalf("without %s: %v", HistoryEnv, err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory available: %v", err)
	}
	if want := filepath.Join(home, ".aurora", "history"); got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}
