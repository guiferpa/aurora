package lsp

import (
	"sort"
	"strings"
)

// Mapper converts between byte offsets in a document and LSP positions.
//
// The protocol counts lines from zero and characters in UTF-16 code units, while the
// lexer reports 1-based lines and byte columns. Any source holding a non-ASCII rune (a
// text like "áé") drifts if byte columns are handed to a client directly, so positions
// are always derived here from the document text and a token's byte offset
// (token.Token.GetCursor).
type Mapper struct {
	text  string
	lines []int // byte offset where each line starts
}

func NewMapper(text string) *Mapper {
	lines := make([]int, 1, strings.Count(text, "\n")+1)
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, i+1)
		}
	}
	return &Mapper{text: text, lines: lines}
}

// Position converts a byte offset into an LSP position, clamped to the document.
func (m *Mapper) Position(offset int) Position {
	offset = m.clamp(offset)
	line := sort.Search(len(m.lines), func(i int) bool { return m.lines[i] > offset }) - 1
	if line < 0 {
		line = 0
	}
	return Position{Line: line, Character: utf16Len(m.text[m.lines[line]:offset])}
}

// Range converts a byte offset and a byte length into an LSP range.
func (m *Mapper) Range(offset, length int) Range {
	if length < 0 {
		length = 0
	}
	return Range{Start: m.Position(offset), End: m.Position(offset + length)}
}

// RangeAt converts a place written the way a compiler names one — a 1-based line and a
// 1-based column counted in bytes — into a range covering the rest of that line.
//
// The rest of the line rather than one character, because what a diagnostic carries is where
// something starts and not how long it is, and a zero-width marker is drawn by most clients as
// nothing at all. Underlining to the end of the line says "here" without claiming a length
// nobody measured.
func (m *Mapper) RangeAt(line, column int) Range {
	if line < 1 || line > len(m.lines) {
		return Range{}
	}
	offset := m.clamp(m.lines[line-1] + max(column-1, 0))
	return Range{Start: m.Position(offset), End: m.Position(m.LineEndOffset(offset))}
}

// Offset converts an LSP position back into a byte offset, clamped to the document.
// Used to find which token sits under the cursor for hover.
func (m *Mapper) Offset(pos Position) int {
	if pos.Line < 0 {
		return 0
	}
	if pos.Line >= len(m.lines) {
		return len(m.text)
	}
	start := m.lines[pos.Line]
	units := 0
	for i, r := range m.text[start:] {
		if units >= pos.Character || r == '\n' {
			return start + i
		}
		units++
		if r > 0xFFFF {
			units++
		}
	}
	return len(m.text)
}

// LineEndOffset returns the byte offset where the line holding offset ends, excluding the
// line break. Used for comments: the lexer emits the "#-" token but drops everything after
// it, so the comment range has to be recovered from the text.
func (m *Mapper) LineEndOffset(offset int) int {
	offset = m.clamp(offset)
	end := strings.IndexByte(m.text[offset:], '\n')
	if end < 0 {
		end = len(m.text)
	} else {
		end += offset
	}
	if end > offset && m.text[end-1] == '\r' {
		end--
	}
	return end
}

// Text returns the document the mapper was built from.
func (m *Mapper) Text() string {
	return m.text
}

func (m *Mapper) clamp(offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(m.text) {
		return len(m.text)
	}
	return offset
}

// utf16Len counts s in UTF-16 code units: runes outside the BMP take two.
func utf16Len(s string) int {
	units := 0
	for _, r := range s {
		units++
		if r > 0xFFFF {
			units++
		}
	}
	return units
}
