package lexer

import (
	"bytes"
	"github.com/guiferpa/aurora/wire/token"
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
		// token.ID
		{[]byte(`abc`), token.ID, []byte("abc"), true},
		{[]byte(`is_true?`), token.ID, []byte("is_true?"), true},
		{[]byte(`e_não?`), token.ID, []byte("e_n"), true},
		{[]byte(`explore->implore?`), token.ID, []byte("explore->implore?"), true},
		{[]byte(`0d?`), token.NUMBER, []byte("0"), true}, // Matches '0' as token.NUMBER
		{[]byte(`Id?`), token.ID, []byte("Id?"), true},
		// token.SEMICOLON
		{[]byte(`;`), token.SEMICOLON, []byte(";"), true},
		// token.COLON
		{[]byte(`:`), token.COLON, []byte(":"), true},
		// token.IF
		{[]byte(`if () {}`), token.IF, []byte("if"), true},
		// token.ELSE
		{[]byte(`else {}`), token.ELSE, []byte("else"), true},
		// token.COMMA
		{[]byte(`,`), token.COMMA, []byte(","), true},
		// token.C_BRK
		{[]byte(`]`), token.C_BRK, []byte("]"), true},
		// token.O_BRK
		{[]byte(`[`), token.O_BRK, []byte("["), true},
		// token.C_CUR_BRK
		{[]byte(`}`), token.C_CUR_BRK, []byte("}"), true},
		// token.O_CUR_BRK
		{[]byte(`{`), token.O_CUR_BRK, []byte("{"), true},
		// token.BRANCH
		{[]byte(`branch [true: 1,]`), token.BRANCH, []byte("branch"), true},
		// token.DEFER
		{[]byte(`defer`), token.DEFER, []byte("defer"), true},
		// "nothing" was a keyword and is an ordinary identifier now
		{[]byte(`nothing`), token.ID, []byte("nothing"), true},
		// "::" was the namespace operator: two ordinary colons now
		{[]byte(`::`), token.COLON, []byte(":"), true},
		// COMMENT
		{[]byte(`#-`), token.COMMENT_LINE, []byte("#-"), true},
		// token.SUB
		{[]byte(`-`), token.SUB, []byte("-"), true},
		// token.SUM
		{[]byte(`+`), token.SUM, []byte("+"), true},
		// token.SMALLER
		{[]byte(`smaller`), token.SMALLER, []byte("smaller"), true},
		// token.BIGGER
		{[]byte(`bigger`), token.BIGGER, []byte("bigger"), true},
		// token.DIFFERENT
		{[]byte(`different`), token.DIFFERENT, []byte("different"), true},
		// token.EQUALS
		{[]byte(`equals`), token.EQUALS, []byte("equals"), true},
		// token.C_PAREN
		{[]byte(`)`), token.C_PAREN, []byte(")"), true},
		// token.O_PAREN
		{[]byte(`(`), token.O_PAREN, []byte("("), true},
		// "as" was an import keyword, then an ordinary identifier, and is a keyword again:
		// it names the shape a value is read with.
		{[]byte(`as`), token.AS, []byte("as"), true},
		// token.SHAPE
		{[]byte(`shape`), token.SHAPE, []byte("shape"), true},
		// token.DOT
		{[]byte(`.`), token.DOT, []byte("."), true},
		// token.ASSIGN
		{[]byte(`=`), token.ASSIGN, []byte("="), true},
		// token.IDENT
		{[]byte(`ident`), token.IDENT, []byte("ident"), true},
		// token.NUMBER
		{[]byte(`1000`), token.NUMBER, []byte("1000"), true},
		{[]byte(`1_000`), token.NUMBER, []byte("1_000"), true},
		{[]byte(`10`), token.NUMBER, []byte("10"), true},
		{[]byte(`9`), token.NUMBER, []byte("9"), true},
		// token.NUMBER - Hexadecimal
		{[]byte(`0xFF`), token.NUMBER, []byte("0xFF"), true},
		{[]byte(`0xff`), token.NUMBER, []byte("0xff"), true},
		{[]byte(`0XFF`), token.NUMBER, []byte("0XFF"), true},
		{[]byte(`0x10`), token.NUMBER, []byte("0x10"), true},
		{[]byte(`0x1A`), token.NUMBER, []byte("0x1A"), true},
		{[]byte(`0xABCD`), token.NUMBER, []byte("0xABCD"), true},
		{[]byte(`0xabcd`), token.NUMBER, []byte("0xabcd"), true},
		{[]byte(`0xAbCd`), token.NUMBER, []byte("0xAbCd"), true},
		{[]byte(`0x0`), token.NUMBER, []byte("0x0"), true},
		{[]byte(`0x00`), token.NUMBER, []byte("0x00"), true},
		// token.WHITESPACE
		{[]byte(`  `), token.WHITESPACE, []byte(`  `), true},
		// token.BREAK_LINE
		{[]byte(`
`), token.BREAK_LINE, []byte(`
`), true},
		// "use" is an import keyword again: it brings a module in under an alias
		{[]byte(`use`), token.USE, []byte("use"), true},
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
