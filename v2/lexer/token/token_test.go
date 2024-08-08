package token

import (
	"bytes"
	"strings"
	"testing"
)

func TestTokenMatchGivenTagRule(t *testing.T) {
	cases := []struct {
		Buffer  *bytes.Buffer
		TagId   string
		Match   string
		Matched bool
	}{
		// ID
		{bytes.NewBufferString(`abc`), ID, "abc", true},
		{bytes.NewBufferString(`is_true?`), ID, "is_true?", true},
		{bytes.NewBufferString(`e_não?`), ID, "e_n", true},
		{bytes.NewBufferString(`explore->implore?`), ID, "explore->implore?", true},
		{bytes.NewBufferString(`0d?`), "", "", false}, // Exception
		{bytes.NewBufferString(`Id?`), ID, "Id?", true},

		// SEMICOLON
		{bytes.NewBufferString(`;`), SEMICOLON, ";", true},

		// COLON
		{bytes.NewBufferString(`:`), COLON, ":", true},

		// IF
		{bytes.NewBufferString(`if () {}`), IF, "if", true},

		// COMMA
		{bytes.NewBufferString(`,`), COMMA, ",", true},

		// C_BRK
		{bytes.NewBufferString(`}`), C_BRK, "}", true},

		// O_BRK
		{bytes.NewBufferString(`{`), O_BRK, "{", true},

		// COMMENT
		{bytes.NewBufferString(`#-`), COMMENT, "#-", true},

		// SUB
		{bytes.NewBufferString(`-`), SUB, "-", true},

		// SUM
		{bytes.NewBufferString(`+`), SUM, "+", true},

		// SMALLER
		{bytes.NewBufferString(`smaller`), SMALLER, "smaller", true},

		// BIGGER
		{bytes.NewBufferString(`bigger`), BIGGER, "bigger", true},

		// DIFFERENT
		{bytes.NewBufferString(`different`), DIFFERENT, "different", true},

		// EQUALS
		{bytes.NewBufferString(`equals`), EQUALS, "equals", true},

		// C_PAREN
		{bytes.NewBufferString(`)`), C_PAREN, ")", true},

		// O_PAREN
		{bytes.NewBufferString(`(`), O_PAREN, "(", true},

		// ASSIGN
		{bytes.NewBufferString(`=`), ASSIGN, "=", true},
	}
	for _, c := range cases {
		matched, tag, match := tokenMatchGivenTagRule(c.Buffer.Bytes())
		if matched != c.Matched {
			t.Errorf("rule matching: param: %s, expected: %v, got: %v", string(c.Buffer.Bytes()), c.Matched, matched)
		}
		if strings.Compare(c.TagId, tag.Id) != 0 {
			t.Errorf("param: %s, expected: %s, got: %s", string(c.Buffer.Bytes()), c.TagId, tag.Id)
		}
		if strings.Compare(c.Match, match) != 0 {
			t.Errorf("param: %s, expected: %s, got: %s", string(c.Buffer.Bytes()), c.Match, match)
		}
	}
}
