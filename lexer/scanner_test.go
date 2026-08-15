package lexer

import (
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
		{"open paren", "(", true, O_PAREN, "("},
		{"close paren", ")", true, C_PAREN, ")"},
		{"open bracket", "[", true, O_BRK, "["},
		{"close bracket", "]", true, C_BRK, "]"},
		{"open curly", "{", true, O_CUR_BRK, "{"},
		{"close curly", "}", true, C_CUR_BRK, "}"},
		{"semicolon", ";", true, SEMICOLON, ";"},
		{"colon", ":", true, COLON, ":"},
		{"comma", ",", true, COMMA, ","},
		{"assign", "=", true, ASSIGN, "="},
		{"plus", "+", true, SUM, "+"},
		{"minus", "-", true, SUB, "-"},
		{"multiply", "*", true, MULT, "*"},
		{"divide", "/", true, DIV, "/"},
		{"exponent", "^", true, EXPO, "^"},

		{"plus with more", "+ 1", true, SUM, "+"},
		{"paren with more", "(abc", true, O_PAREN, "("},

		{"comment", "#-", true, COMMENT_LINE, "#-"},
		{"comment with text", "#- this is a comment", true, COMMENT_LINE, "#-"},

		// "::" was the namespace operator and is two ordinary colons now
		{"colon colon", "::", true, COLON, ":"},
		{"colon then colon", ": :", true, COLON, ":"},

		{"keyword if", "if", true, IF, "if"},
		{"keyword else", "else", true, ELSE, "else"},
		{"keyword ident", "ident", true, IDENT, "ident"},
		{"keyword branch", "branch", true, BRANCH, "branch"},
		{"keyword defer", "defer", true, DEFER, "defer"},
		{"keyword printb", "printb", true, PRINTB, "printb"},
		{"keyword printc", "printc", true, PRINTC, "printc"},
		{"keyword printd", "printd", true, PRINTD, "printd"},
		// "print" and "echo" were the old names and are ordinary identifiers now
		{"printb is an identifier", "print", true, ID, "print"},
		{"echo is an identifier", "echo", true, ID, "echo"},
		{"keyword true", "true", true, TRUE, "true"},
		{"keyword false", "false", true, FALSE, "false"},
		{"keyword equals", "equals", true, EQUALS, "equals"},
		{"keyword different", "different", true, DIFFERENT, "different"},
		{"keyword bigger", "bigger", true, BIGGER, "bigger"},
		{"keyword smaller", "smaller", true, SMALLER, "smaller"},
		{"keyword or", "or", true, OR, "or"},
		{"keyword and", "and", true, AND, "and"},
		{"keyword head", "head", true, HEAD, "head"},
		{"keyword tail", "tail", true, TAIL, "tail"},
		{"keyword push", "push", true, PUSH, "push"},
		{"keyword pull", "pull", true, PULL, "pull"},
		{"keyword feed", "feed", true, FEED, "feed"},
		{"keyword assert", "assert", true, ASSERT, "assert"},
		{"keyword struct", "struct", true, STRUCT, "struct"},
		// "as" was an import keyword, became an ordinary identifier when namespaces were
		// rolled back, and is a keyword again: it names the shape a value is read with.
		{"keyword as", "as", true, AS, "as"},
		// "use" was the other import keyword and stays an ordinary identifier.
		{"use is an identifier", "use", true, ID, "use"},

		{"if with space", "if x", true, IF, "if"},
		{"if with paren", "if(", true, IF, "if"},

		{"simple id", "foo", true, ID, "foo"},
		{"id with digits", "foo123", true, ID, "foo123"},
		{"id with underscore", "my_var", true, ID, "my_var"},
		{"longer id", "myLongVariableName", true, ID, "myLongVariableName"},
		{"if prefix", "iffy", true, ID, "iffy"},
		{"else prefix", "elsewhere", true, ID, "elsewhere"},
		{"true prefix", "trueish", true, ID, "trueish"},
		{"nothing is an ordinary identifier now", "nothing", true, ID, "nothing"},
		{"nothing prefix", "nothingish", true, ID, "nothingish"},

		{"uppercase id", "Foo", true, ID, "Foo"},
		{"uppercase with digits", "Foo123", true, ID, "Foo123"},
		{"all caps", "FOO", true, ID, "FOO"},
		{"mixed case", "MyClass", true, ID, "MyClass"},

		{"single digit", "0", true, NUMBER, "0"},
		{"multi digit", "123", true, NUMBER, "123"},
		{"number multiple underscores", "1_000_000", true, NUMBER, "1_000_000"},

		{"number then space", "123 ", true, NUMBER, "123"},
		{"number then plus", "123+", true, NUMBER, "123"},
		{"number then paren", "123)", true, NUMBER, "123"},

		{"hex lowercase", "0xff", true, NUMBER, "0xff"},
		{"hex uppercase", "0xFF", true, NUMBER, "0xFF"},
		{"hex capital X", "0XFF", true, NUMBER, "0XFF"},
		{"hex single digit", "0x0", true, NUMBER, "0x0"},
		{"hex long", "0xABCDEF", true, NUMBER, "0xABCDEF"},
		{"hex mixed case", "0xAbCd", true, NUMBER, "0xAbCd"},

		{"empty string", `""`, true, STRING, `""`},
		{"simple string", `"hello"`, true, STRING, `"hello"`},
		{"string with spaces", `"hello world"`, true, STRING, `"hello world"`},
		{"string with numbers", `"abc123"`, true, STRING, `"abc123"`},

		{"string then more", `"hello" world`, true, STRING, `"hello"`},

		{"single space", " ", true, WHITESPACE, " "},
		{"multiple spaces", "   ", true, WHITESPACE, "   "},
		{"spaces then text", "  abc", true, WHITESPACE, "  "},

		{"newline", "\n", true, BREAK_LINE, "\n"},
		{"carriage return", "\r", true, BREAK_LINE, "\r"},
		{"newline then text", "\nabc", true, BREAK_LINE, "\n"},

		{"empty input", "", false, "", ""},

		{"unknown char @", "@", false, "", ""},
		{"unknown char $", "$", false, "", ""},
		{"unknown char %", "%", false, "", ""},
		{"unknown char &", "&", false, "", ""},
		{"unknown char ~", "~", false, "", ""},

		{"id with dash", "my-var", true, ID, "my-var"},
		{"id with question mark", "my?var", true, ID, "my?var"},
		{"id with exclamation mark", "my!var", true, ID, "my!var"},
		{"id with greater than", "my>var", true, ID, "my>var"},
		{"id with less than", "my<var", true, ID, "my<var"},
		{"id with greater than or equal to", "my>=var", false, ID, "my>=var"},
		{"id with less than or equal to", "my<=var", false, ID, "my<=var"},
		{"id with not equal to", "my!=var", false, ID, "my!=var"},
		{"id with arrow symbol", "my->var", true, ID, "my->var"},
		{"id with inverted arrow symbol", "my<-var", true, ID, "my<-var"},
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
		{"if", IF, 2},
		{"iff", ID, 3},
		{"if_then", ID, 7},
		{"else", ELSE, 4},
		{"elsewhere", ID, 9},
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

// struct and as are directives for whoever writes the source: they name the fields of a run
// of tapes and say which shape a value is read with. `as` was a keyword before namespaces
// were rolled back, and comes back here for that.
func TestScanStructDirectives(t *testing.T) {
	cases := []struct {
		source string
		want   []string
	}{
		{source: "struct Point { x, y };", want: []string{STRUCT, ID, O_CUR_BRK, ID, COMMA, ID, C_CUR_BRK, SEMICOLON}},
		{source: "p.x;", want: []string{ID, DOT, ID, SEMICOLON}},
		{source: "feed(0) as Point;", want: []string{FEED, O_PAREN, NUMBER, C_PAREN, AS, ID, SEMICOLON}},
	}

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			tokens, err := New(NewLexerOptions{}).GetFilledTokens([]byte(tc.source))
			if err != nil {
				t.Fatalf("lexer: %v", err)
			}

			got := make([]string, 0, len(tokens))
			for _, token := range tokens {
				if id := token.GetTag().Id; id != EOF {
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
	tokens, err := New(NewLexerOptions{}).GetFilledTokens([]byte("point.x"))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	if got := string(tokens[0].GetMatch()); got != "point" {
		t.Errorf("first token is %q, want %q", got, "point")
	}
}
