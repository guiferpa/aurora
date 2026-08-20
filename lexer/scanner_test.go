package lexer

import (
	"github.com/guiferpa/aurora/wire/token"
	"testing"
)

func TestScanToken(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantOk   bool
		wantTag  string
		wantText string
	}{
		{"open paren", "(", true, token.O_PAREN, "("},
		{"close paren", ")", true, token.C_PAREN, ")"},
		{"open bracket", "[", true, token.O_BRK, "["},
		{"close bracket", "]", true, token.C_BRK, "]"},
		{"open curly", "{", true, token.O_CUR_BRK, "{"},
		{"close curly", "}", true, token.C_CUR_BRK, "}"},
		{"semicolon", ";", true, token.SEMICOLON, ";"},
		{"colon", ":", true, token.COLON, ":"},
		{"comma", ",", true, token.COMMA, ","},
		{"assign", "=", true, token.ASSIGN, "="},
		{"plus", "+", true, token.SUM, "+"},
		{"minus", "-", true, token.SUB, "-"},
		{"multiply", "*", true, token.MULT, "*"},
		{"divide", "/", true, token.DIV, "/"},
		{"exponent", "^", true, token.EXPO, "^"},

		{"plus with more", "+ 1", true, token.SUM, "+"},
		{"paren with more", "(abc", true, token.O_PAREN, "("},

		{"comment", "#-", true, token.COMMENT_LINE, "#-"},
		{"comment with text", "#- this is a comment", true, token.COMMENT_LINE, "#-"},

		// "::" was the namespace operator and is two ordinary colons now
		{"colon colon", "::", true, token.COLON, ":"},
		{"colon then colon", ": :", true, token.COLON, ":"},

		{"keyword if", "if", true, token.IF, "if"},
		{"keyword else", "else", true, token.ELSE, "else"},
		{"keyword ident", "ident", true, token.IDENT, "ident"},
		{"keyword branch", "branch", true, token.BRANCH, "branch"},
		{"keyword defer", "defer", true, token.DEFER, "defer"},
		{"keyword printb", "printb", true, token.PRINTB, "printb"},
		{"keyword printc", "printc", true, token.PRINTC, "printc"},
		{"keyword printd", "printd", true, token.PRINTD, "printd"},
		// "print" and "echo" were the old names and are ordinary identifiers now
		{"printb is an identifier", "print", true, token.ID, "print"},
		{"echo is an identifier", "echo", true, token.ID, "echo"},
		{"keyword true", "true", true, token.TRUE, "true"},
		{"keyword false", "false", true, token.FALSE, "false"},
		{"keyword equals", "equals", true, token.EQUALS, "equals"},
		{"keyword different", "different", true, token.DIFFERENT, "different"},
		{"keyword bigger", "bigger", true, token.BIGGER, "bigger"},
		{"keyword smaller", "smaller", true, token.SMALLER, "smaller"},
		{"keyword or", "or", true, token.OR, "or"},
		{"keyword and", "and", true, token.AND, "and"},
		{"keyword head", "head", true, token.HEAD, "head"},
		{"keyword tail", "tail", true, token.TAIL, "tail"},
		{"keyword push", "push", true, token.PUSH, "push"},
		{"keyword pull", "pull", true, token.PULL, "pull"},
		{"keyword feed", "feed", true, token.FEED, "feed"},
		{"keyword assert", "assert", true, token.ASSERT, "assert"},
		{"keyword shape", "shape", true, token.SHAPE, "shape"},
		// "as" was an import keyword, became an ordinary identifier when namespaces were
		// rolled back, and is a keyword again: it names the shape a value is read with.
		{"keyword as", "as", true, token.AS, "as"},
		// "use" was the other import keyword, became an ordinary identifier when namespaces
		// were rolled back, and is a keyword again: it brings a module in.
		{"keyword use", "use", true, token.USE, "use"},
		{"use prefix", "useful", true, token.ID, "useful"},

		{"if with space", "if x", true, token.IF, "if"},
		{"if with paren", "if(", true, token.IF, "if"},

		{"simple id", "foo", true, token.ID, "foo"},
		{"id with digits", "foo123", true, token.ID, "foo123"},
		{"id with underscore", "my_var", true, token.ID, "my_var"},
		{"longer id", "myLongVariableName", true, token.ID, "myLongVariableName"},
		{"if prefix", "iffy", true, token.ID, "iffy"},
		{"else prefix", "elsewhere", true, token.ID, "elsewhere"},
		{"true prefix", "trueish", true, token.ID, "trueish"},
		{"nothing is an ordinary identifier now", "nothing", true, token.ID, "nothing"},
		{"nothing prefix", "nothingish", true, token.ID, "nothingish"},

		{"uppercase id", "Foo", true, token.ID, "Foo"},
		{"uppercase with digits", "Foo123", true, token.ID, "Foo123"},
		{"all caps", "FOO", true, token.ID, "FOO"},
		{"mixed case", "MyClass", true, token.ID, "MyClass"},

		{"single digit", "0", true, token.NUMBER, "0"},
		{"multi digit", "123", true, token.NUMBER, "123"},
		{"number multiple underscores", "1_000_000", true, token.NUMBER, "1_000_000"},

		{"number then space", "123 ", true, token.NUMBER, "123"},
		{"number then plus", "123+", true, token.NUMBER, "123"},
		{"number then paren", "123)", true, token.NUMBER, "123"},

		{"hex lowercase", "0xff", true, token.NUMBER, "0xff"},
		{"hex uppercase", "0xFF", true, token.NUMBER, "0xFF"},
		{"hex capital X", "0XFF", true, token.NUMBER, "0XFF"},
		{"hex single digit", "0x0", true, token.NUMBER, "0x0"},
		{"hex long", "0xABCDEF", true, token.NUMBER, "0xABCDEF"},
		{"hex mixed case", "0xAbCd", true, token.NUMBER, "0xAbCd"},

		{"empty string", `""`, true, token.STRING, `""`},
		{"simple string", `"hello"`, true, token.STRING, `"hello"`},
		{"string with spaces", `"hello world"`, true, token.STRING, `"hello world"`},
		{"string with numbers", `"abc123"`, true, token.STRING, `"abc123"`},

		{"string then more", `"hello" world`, true, token.STRING, `"hello"`},

		{"single space", " ", true, token.WHITESPACE, " "},
		{"multiple spaces", "   ", true, token.WHITESPACE, "   "},
		{"spaces then text", "  abc", true, token.WHITESPACE, "  "},

		{"newline", "\n", true, token.BREAK_LINE, "\n"},
		{"carriage return", "\r", true, token.BREAK_LINE, "\r"},
		{"newline then text", "\nabc", true, token.BREAK_LINE, "\n"},

		{"empty input", "", false, "", ""},

		{"unknown char @", "@", false, "", ""},
		{"unknown char $", "$", false, "", ""},
		{"unknown char %", "%", false, "", ""},
		{"unknown char &", "&", false, "", ""},
		{"unknown char ~", "~", false, "", ""},

		{"id with dash", "my-var", true, token.ID, "my-var"},
		{"id with question mark", "my?var", true, token.ID, "my?var"},
		{"id with exclamation mark", "my!var", true, token.ID, "my!var"},
		{"id with greater than", "my>var", true, token.ID, "my>var"},
		{"id with less than", "my<var", true, token.ID, "my<var"},
		{"id with greater than or equal to", "my>=var", false, token.ID, "my>=var"},
		{"id with less than or equal to", "my<=var", false, token.ID, "my<=var"},
		{"id with not equal to", "my!=var", false, token.ID, "my!=var"},
		{"id with arrow symbol", "my->var", true, token.ID, "my->var"},
		{"id with inverted arrow symbol", "my<-var", true, token.ID, "my<-var"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			matched, tag, match := ScanToken([]byte(c.input))

			if matched != c.wantOk {
				t.Errorf("ScanToken(%q) matched = %v, want %v", c.input, matched, c.wantOk)
				return
			}

			if !c.wantOk {
				return
			}

			if tag.Id != c.wantTag {
				t.Errorf("ScanToken(%q) tag.Id = %q, want %q", c.input, tag.Id, c.wantTag)
			}

			if string(match) != c.wantText {
				t.Errorf("ScanToken(%q) match = %q, want %q", c.input, string(match), c.wantText)
			}
		})
	}
}

func TestIsLowercaseLetter(t *testing.T) {
	for c := byte('a'); c <= 'z'; c++ {
		if !isLowercaseLetter(c) {
			t.Errorf("isLowercaseLetter(%q) = false, want true", c)
		}
	}
	for c := byte('A'); c <= 'Z'; c++ {
		if isLowercaseLetter(c) {
			t.Errorf("isLowercaseLetter(%q) = true, want false", c)
		}
	}
}

func TestIsUppercaseLetter(t *testing.T) {
	for c := byte('A'); c <= 'Z'; c++ {
		if !isUppercaseLetter(c) {
			t.Errorf("isUppercaseLetter(%q) = false, want true", c)
		}
	}
	for c := byte('a'); c <= 'z'; c++ {
		if isUppercaseLetter(c) {
			t.Errorf("isUppercaseLetter(%q) = true, want false", c)
		}
	}
}

func TestIsDigit(t *testing.T) {
	for c := byte('0'); c <= '9'; c++ {
		if !isDigit(c) {
			t.Errorf("isDigit(%q) = false, want true", c)
		}
	}
	if isDigit('a') {
		t.Error("isDigit('a') = true, want false")
	}
}

func TestIsHexDigit(t *testing.T) {
	valid := "0123456789abcdefABCDEF"
	for _, c := range valid {
		if !isHexDigit(byte(c)) {
			t.Errorf("isHexDigit(%q) = false, want true", c)
		}
	}
	invalid := "ghijGHIJ@#$"
	for _, c := range invalid {
		if isHexDigit(byte(c)) {
			t.Errorf("isHexDigit(%q) = true, want false", c)
		}
	}
}

func TestIsIdentChar(t *testing.T) {
	valid := "abcABC123_-?!><"
	for _, c := range valid {
		if !isIdentChar(byte(c)) {
			t.Errorf("isIdentChar(%q) = false, want true", c)
		}
	}
	invalid := "+*/(){}[]"
	for _, c := range invalid {
		if isIdentChar(byte(c)) {
			t.Errorf("isIdentChar(%q) = true, want false", c)
		}
	}
}

func TestScanWord(t *testing.T) {
	cases := []struct {
		input   string
		wantTag string
		wantLen int
	}{
		{"if", token.IF, 2},
		{"iff", token.ID, 3},
		{"if_then", token.ID, 7},
		{"else", token.ELSE, 4},
		{"elsewhere", token.ID, 9},
	}

	for _, c := range cases {
		matched, tag, match := scanWord([]byte(c.input))
		if !matched {
			t.Errorf("scanWord(%q) didn't match", c.input)
			continue
		}
		if tag.Id != c.wantTag {
			t.Errorf("scanWord(%q) tag = %q, want %q", c.input, tag.Id, c.wantTag)
		}
		if len(match) != c.wantLen {
			t.Errorf("scanWord(%q) len = %d, want %d", c.input, len(match), c.wantLen)
		}
	}
}

func TestUnderscoreIdentifier(t *testing.T) {
	input := "_private"
	matched, tag, match := ScanToken([]byte(input))

	if matched {
		t.Logf("ScanToken(%q) matched with tag=%q, match=%q", input, tag.Id, string(match))
	} else {
		t.Logf("ScanToken(%q) did not match (underscore-start not supported)", input)
	}
}

// shape and as are keywords that declare rather than do: they name the fields of a run
// of tapes and say which shape a value is read with. `as` was a keyword before namespaces
// were rolled back, and comes back here for that.
func TestScanShapeDeclarations(t *testing.T) {
	cases := []struct {
		source string
		want   []string
	}{
		{source: "shape Point { x, y };", want: []string{token.SHAPE, token.ID, token.O_CUR_BRK, token.ID, token.COMMA, token.ID, token.C_CUR_BRK, token.SEMICOLON}},
		{source: "p.x;", want: []string{token.ID, token.DOT, token.ID, token.SEMICOLON}},
		{source: "feed(0) as Point;", want: []string{token.FEED, token.O_PAREN, token.NUMBER, token.C_PAREN, token.AS, token.ID, token.SEMICOLON}},
	}

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			tokens, err := New().GetFilledTokens([]byte(tc.source))
			if err != nil {
				t.Fatalf("lexer: %v", err)
			}

			got := make([]string, 0, len(tokens))
			for _, tk := range tokens {
				if id := tk.GetTag().Id; id != token.EOF {
					got = append(got, id)
				}
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("token %d is %s, want %s", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// A dot is its own token and never part of a name, so p.x is three tokens and not one.
func TestDotIsNotPartOfAName(t *testing.T) {
	tokens, err := New().GetFilledTokens([]byte("point.x"))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	if got := string(tokens[0].GetMatch()); got != "point" {
		t.Errorf("first token is %q, want %q", got, "point")
	}
}
