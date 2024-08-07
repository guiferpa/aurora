package token

import (
	"bytes"
	"strings"
	"testing"
)

func TestTokenMatchGivenTagRule(t *testing.T) {
	cases := []struct {
		Buffer *bytes.Buffer
		TagId  string
		Match  string
	}{
		// ID
		{bytes.NewBufferString(`abc`), ID, "abc"},
		{bytes.NewBufferString(`is_true?`), ID, "is_true?"},
		{bytes.NewBufferString(`e_não?`), ID, "e_n"},
		{bytes.NewBufferString(`explore->implore?`), ID, "explore->implore?"},
		{bytes.NewBufferString(`0d?`), "", ""}, // Exception
		{bytes.NewBufferString(`Id?`), ID, "Id?"},

		// SEMICOLON
		{bytes.NewBufferString(`;`), SEMICOLON, ";"},

		// COLON
		{bytes.NewBufferString(`:`), COLON, ":"},

		// IF
		{bytes.NewBufferString(`if () {}`), IF, "if"},

		// COMMA
		{bytes.NewBufferString(`,`), COMMA, ","},

		// C_BRK
		{bytes.NewBufferString(`}`), C_BRK, "}"},

		// O_BRK
		{bytes.NewBufferString(`{`), O_BRK, "{"},

		// COMMENT
		{bytes.NewBufferString(`--`), COMMENT, "--"},
	}
	for _, c := range cases {
		_, tag, match := tokenMatchGivenTagRule(c.Buffer.Bytes())
		if strings.Compare(c.TagId, tag.Id) != 0 {
			t.Errorf("param: %s, expected: %s, got: %s", string(c.Buffer.Bytes()), c.TagId, tag.Id)
		}
		if strings.Compare(c.Match, match) != 0 {
			t.Errorf("expected: %s, got: %s", c.Match, match)
		}
	}
}
