package lsp

import "testing"

func TestMapperPosition(t *testing.T) {
	const source = "ident a = 1;\nident b = \"áé\";\nprint b;\n"

	cases := []struct {
		name      string
		offset    int
		line      int
		character int
	}{
		{name: "start of document", offset: 0, line: 0, character: 0},
		{name: "inside first line", offset: 6, line: 0, character: 6},
		{name: "start of second line", offset: 13, line: 1, character: 0},
		// Offset 28 sits 15 bytes into line 2, but only 13 UTF-16 units in: "á" and "é"
		// take two bytes each and one unit each. A byte column would report 15 here.
		{name: "after multi-byte runes", offset: 28, line: 1, character: 13},
		{name: "past the end is clamped", offset: 9999, line: 3, character: 0},
	}

	mapper := NewMapper(source)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapper.Position(tc.offset)
			if got.Line != tc.line || got.Character != tc.character {
				t.Errorf("Position(%d) = %d:%d, want %d:%d", tc.offset, got.Line, got.Character, tc.line, tc.character)
			}
		})
	}
}

func TestMapperRange(t *testing.T) {
	mapper := NewMapper("ident a = 1;\nprint a;\n")
	got := mapper.Range(13, 5) // "print"
	if got.Start.Line != 1 || got.Start.Character != 0 {
		t.Errorf("start = %d:%d, want 1:0", got.Start.Line, got.Start.Character)
	}
	if got.End.Line != 1 || got.End.Character != 5 {
		t.Errorf("end = %d:%d, want 1:5", got.End.Line, got.End.Character)
	}
}

func TestMapperOffsetRoundTrip(t *testing.T) {
	const source = "ident a = \"áé\";\nprint a;\n"
	mapper := NewMapper(source)

	for _, offset := range []int{0, 6, 10, 16, 20} {
		pos := mapper.Position(offset)
		if got := mapper.Offset(pos); got != offset {
			t.Errorf("Offset(Position(%d)) = %d, want %d", offset, got, offset)
		}
	}
}

func TestMapperOffsetClampsBeyondLine(t *testing.T) {
	mapper := NewMapper("ab\ncd\n")
	// Character past the end of the line must stop at the line break, not spill over.
	if got := mapper.Offset(Position{Line: 0, Character: 99}); got != 2 {
		t.Errorf("Offset = %d, want 2", got)
	}
	if got := mapper.Offset(Position{Line: 99, Character: 0}); got != 6 {
		t.Errorf("Offset past last line = %d, want 6", got)
	}
}

func TestMapperLineEndOffset(t *testing.T) {
	cases := []struct {
		name   string
		source string
		offset int
		want   int
	}{
		{name: "line with break", source: "abc\ndef\n", offset: 1, want: 3},
		{name: "last line without break", source: "abc\ndef", offset: 5, want: 7},
		{name: "carriage return is excluded", source: "abc\r\ndef", offset: 0, want: 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewMapper(tc.source).LineEndOffset(tc.offset); got != tc.want {
				t.Errorf("LineEndOffset(%d) = %d, want %d", tc.offset, got, tc.want)
			}
		})
	}
}
