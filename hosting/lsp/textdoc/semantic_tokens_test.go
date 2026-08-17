package textdoc

import (
	"reflect"
	"testing"
)

// decoded is one 5-tuple, easier to read in a failure than a flat slice.
type decoded struct {
	deltaLine, deltaChar, length, tokenType, modifiers uint
}

func decode(data []uint) []decoded {
	tokens := make([]decoded, 0, len(data)/5)
	for i := 0; i+4 < len(data); i += 5 {
		tokens = append(tokens, decoded{data[i], data[i+1], data[i+2], data[i+3], data[i+4]})
	}
	return tokens
}

func TestSemanticTokens(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   []decoded
	}{
		{
			name:   "declaration, operator and number",
			source: "ident a = 1;",
			want: []decoded{
				{0, 0, 5, SemanticKeyword, 0},                            // ident
				{0, 6, 1, SemanticVariable, SemanticModifierDeclaration}, // a
				{0, 2, 1, SemanticOperator, 0},                           // =
				{0, 2, 1, SemanticNumber, 0},                             // 1
			},
		},
		{
			name:   "call is a function, plain use is a variable",
			source: "printb sum(a);",
			want: []decoded{
				{0, 0, 6, SemanticKeyword, 0},  // printb
				{0, 7, 3, SemanticFunction, 0}, // sum(
				{0, 4, 1, SemanticVariable, 0}, // a
			},
		},
		{
			name:   "comment covers the rest of the line",
			source: "#- a comment\nident a = 1;",
			want: []decoded{
				{0, 0, 12, SemanticComment, 0},
				{1, 0, 5, SemanticKeyword, 0},
				{0, 6, 1, SemanticVariable, SemanticModifierDeclaration},
				{0, 2, 1, SemanticOperator, 0},
				{0, 2, 1, SemanticNumber, 0},
			},
		},
		{
			name:   "reel literal",
			source: `printc "hi";`,
			want: []decoded{
				{0, 0, 6, SemanticKeyword, 0},
				{0, 7, 4, SemanticString, 0},
			},
		},
		{
			name:   "multi-byte runes are measured in utf-16 units",
			source: `printc "áé";`,
			want: []decoded{
				{0, 0, 6, SemanticKeyword, 0},
				{0, 7, 4, SemanticString, 0}, // "áé" = 6 bytes, 4 UTF-16 units
			},
		},
		{
			name:   "defer and feed",
			source: "ident f = defer { feed(0); };",
			want: []decoded{
				{0, 0, 5, SemanticKeyword, 0},
				{0, 6, 1, SemanticVariable, SemanticModifierDeclaration},
				{0, 2, 1, SemanticOperator, 0},
				{0, 2, 5, SemanticKeyword, 0}, // defer
				{0, 8, 4, SemanticKeyword, 0}, // feed
				{0, 5, 1, SemanticNumber, 0},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decode(SemanticTokensFor(tc.source))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got:\n%v\nwant:\n%v", got, tc.want)
			}
		})
	}
}

// A file mid-edit must keep its colors: the lexer returns the tokens it managed to read
// along with the error, and coloring uses them.
func TestSemanticTokensOnBrokenSource(t *testing.T) {
	data := SemanticTokensFor("ident a = 1;\nident b = @@@;")
	tokens := decode(data)
	if len(tokens) < 5 {
		t.Fatalf("expected the valid prefix to still be colored, got %d tokens", len(tokens))
	}
	if tokens[0] != (decoded{0, 0, 5, SemanticKeyword, 0}) {
		t.Errorf("first token = %v, want the ident keyword", tokens[0])
	}
}

func TestSemanticTokensEmptyDocument(t *testing.T) {
	if data := SemanticTokensFor(""); len(data) != 0 {
		t.Errorf("expected no tokens, got %v", data)
	}
}

// Every entry must be a full 5-tuple, or clients misread the whole stream.
func TestSemanticTokensDataIsWellFormed(t *testing.T) {
	data := SemanticTokensFor("ident a = 1;\n#- note\nprintb a;\n")
	if len(data)%5 != 0 {
		t.Fatalf("data length %d is not a multiple of 5", len(data))
	}
}
