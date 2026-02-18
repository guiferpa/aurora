package lexer

import (
	"bytes"
	"strings"
	"testing"
)

func TestMatchToken(t *testing.T) {
	cases := []struct {
		Buffer  []byte
		TagId   string
		Match   []byte
		Matched bool
	}{
		// ID
		{[]byte(`abc`), ID, []byte("abc"), true},
		{[]byte(`is_true?`), ID, []byte("is_true?"), true},
		{[]byte(`e_não?`), ID, []byte("e_n"), true},
		{[]byte(`explore->implore?`), ID, []byte("explore->implore?"), true},
		{[]byte(`0d?`), NUMBER, []byte("0"), true}, // Matches '0' as NUMBER
		{[]byte(`Id?`), ID, []byte("Id?"), true},
		// SEMICOLON
		{[]byte(`;`), SEMICOLON, []byte(";"), true},
		// COLON
		{[]byte(`:`), COLON, []byte(":"), true},
		// IF
		{[]byte(`if () {}`), IF, []byte("if"), true},
		// ELSE
		{[]byte(`else {}`), ELSE, []byte("else"), true},
		// COMMA
		{[]byte(`,`), COMMA, []byte(","), true},
		// C_BRK
		{[]byte(`]`), C_BRK, []byte("]"), true},
		// O_BRK
		{[]byte(`[`), O_BRK, []byte("["), true},
		// C_CUR_BRK
		{[]byte(`}`), C_CUR_BRK, []byte("}"), true},
		// O_CUR_BRK
		{[]byte(`{`), O_CUR_BRK, []byte("{"), true},
		// BRANCH
		{[]byte(`branch [true: 1,]`), BRANCH, []byte("branch"), true},
		// DEFER
		{[]byte(`defer`), DEFER, []byte("defer"), true},
		// NOTHING
		{[]byte(`nothing`), NOTHING, []byte("nothing"), true},
		{[]byte(`nothing;`), NOTHING, []byte("nothing"), true},
		// COMMENT
		{[]byte(`#-`), COMMENT_LINE, []byte("#-"), true},
		// SUB
		{[]byte(`-`), SUB, []byte("-"), true},
		// SUM
		{[]byte(`+`), SUM, []byte("+"), true},
		// SMALLER
		{[]byte(`smaller`), SMALLER, []byte("smaller"), true},
		// BIGGER
		{[]byte(`bigger`), BIGGER, []byte("bigger"), true},
		// DIFFERENT
		{[]byte(`different`), DIFFERENT, []byte("different"), true},
		// EQUALS
		{[]byte(`equals`), EQUALS, []byte("equals"), true},
		// C_PAREN
		{[]byte(`)`), C_PAREN, []byte(")"), true},
		// O_PAREN
		{[]byte(`(`), O_PAREN, []byte("("), true},
		// ASSIGN
		{[]byte(`=`), ASSIGN, []byte("="), true},
		// IDENT
		{[]byte(`ident`), IDENT, []byte("ident"), true},
		// NUMBER
		{[]byte(`1000`), NUMBER, []byte("1000"), true},
		{[]byte(`1_000`), NUMBER, []byte("1_000"), true},
		{[]byte(`10`), NUMBER, []byte("10"), true},
		{[]byte(`9`), NUMBER, []byte("9"), true},
		// NUMBER - Hexadecimal
		{[]byte(`0xFF`), NUMBER, []byte("0xFF"), true},
		{[]byte(`0xff`), NUMBER, []byte("0xff"), true},
		{[]byte(`0XFF`), NUMBER, []byte("0XFF"), true},
		{[]byte(`0x10`), NUMBER, []byte("0x10"), true},
		{[]byte(`0x1A`), NUMBER, []byte("0x1A"), true},
		{[]byte(`0xABCD`), NUMBER, []byte("0xABCD"), true},
		{[]byte(`0xabcd`), NUMBER, []byte("0xabcd"), true},
		{[]byte(`0xAbCd`), NUMBER, []byte("0xAbCd"), true},
		{[]byte(`0x0`), NUMBER, []byte("0x0"), true},
		{[]byte(`0x00`), NUMBER, []byte("0x00"), true},
		// WHITESPACE
		{[]byte(`  `), WHITESPACE, []byte(`  `), true},
		// BREAK_LINE
		{[]byte(`
`), BREAK_LINE, []byte(`
`), true},
	}
	for _, c := range cases {
		matched, tag, match := MatchToken(c.Buffer)
		if matched != c.Matched {
			t.Errorf("rule matching: param: %s, expected: %v, got: %v", string(c.Buffer), c.Matched, matched)
		}
		if strings.Compare(c.TagId, tag.Id) != 0 {
			t.Errorf("param: %s, expected: %v, got: %v", string(c.Buffer), c.TagId, tag.Id)
		}
		if !bytes.Equal(c.Match, match) {
			t.Errorf("param: %s, expected: %v, got: %v", string(c.Buffer), c.Match, match)
		}
	}
}
