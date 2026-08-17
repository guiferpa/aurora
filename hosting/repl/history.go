package repl

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// HistoryLimit is the number of entries kept in the history file.
	HistoryLimit = 1000

	// historyRewriteAt is the size that triggers compaction back to HistoryLimit.
	// Compacting only past twice the limit keeps Append a plain O_APPEND write most of the time.
	historyRewriteAt = 2 * HistoryLimit

	historyDirName  = ".aurora"
	historyFileName = "history"

	// HistoryEnv overrides the history location (useful for tests and for opting out with a temp path).
	HistoryEnv = "AURORA_HISTORY"
)

// History is the REPL command history. It is shared by every project (one file in the
// user's home) and persisted line by line, as the shell does.
type History struct {
	path    string
	entries []string
	limit   int
}

// DefaultHistoryPath returns $AURORA_HISTORY when set, otherwise ~/.aurora/history.
func DefaultHistoryPath() (string, error) {
	if path := os.Getenv(HistoryEnv); path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, historyDirName, historyFileName), nil
}

// LoadHistory reads the history stored at path. It never fails: a missing or unreadable
// file yields an empty history, since history is a convenience and must not break the REPL.
// An empty path yields a memory-only history.
func LoadHistory(path string) *History {
	h := &History{path: path, entries: make([]string, 0), limit: HistoryLimit}

	fd, err := os.Open(path)
	if err != nil {
		return h
	}
	defer func() { _ = fd.Close() }()

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		h.entries = append(h.entries, line)
	}

	if len(h.entries) > historyRewriteAt {
		h.trim()
		_ = h.compact()
		return h
	}
	h.trim()
	return h
}

// Append records line as the most recent entry and persists it. Empty lines and a repeat
// of the previous entry are ignored. Errors are returned but are never fatal for the caller.
func (h *History) Append(line string) error {
	if h == nil || strings.TrimSpace(line) == "" {
		return nil
	}
	if n := len(h.entries); n > 0 && h.entries[n-1] == line {
		return nil
	}

	h.entries = append(h.entries, line)

	if h.path == "" {
		h.trim()
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(h.path), 0o700); err != nil {
		return err
	}
	// 0600: history holds the user's own source lines.
	fd, err := os.OpenFile(h.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(fd, line); err != nil {
		_ = fd.Close()
		return err
	}
	if err := fd.Close(); err != nil {
		return err
	}

	if len(h.entries) > historyRewriteAt {
		h.trim()
		return h.compact()
	}
	return nil
}

// Len returns the number of entries, oldest first.
func (h *History) Len() int {
	if h == nil {
		return 0
	}
	return len(h.entries)
}

// At returns the entry at index i (0 = oldest), or "" when out of range.
func (h *History) At(i int) string {
	if h == nil || i < 0 || i >= len(h.entries) {
		return ""
	}
	return h.entries[i]
}

// trim keeps only the most recent limit entries in memory.
func (h *History) trim() {
	if len(h.entries) > h.limit {
		h.entries = append([]string(nil), h.entries[len(h.entries)-h.limit:]...)
	}
}

// compact rewrites the file with the entries currently in memory, via a temp file and rename.
func (h *History) compact() error {
	if h.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(h.path), 0o700); err != nil {
		return err
	}

	tmp := h.path + ".tmp"
	fd, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(fd)
	for _, entry := range h.entries {
		if _, err := fmt.Fprintln(w, entry); err != nil {
			_ = fd.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		_ = fd.Close()
		return err
	}
	if err := fd.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, h.path)
}
