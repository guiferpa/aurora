package token

import (
	"testing"
)

func TestGetTag(t *testing.T) {
	cases := []struct {
		Tag Tag
		Param    string
	}{
		{tIdent, IDENT},
		{tAssign, ASSIGN},
		{tOParen, O_PAREN},
		{tCParen, C_PAREN},
		{tEquals, EQUALS},
		{tDifferent, DIFFERENT},
		{tBigger, BIGGER},
		{tSmaller, SMALLER},
		{tSum, SUM},
		{tSub, SUB},
		{tComment, COMMENT},
		{tOBrk, O_BRK},
		{tCBrk, C_BRK},
		{tComma, COMMA},
		{tIf, IF},
		{tColon, COLON},
		{tSemicolon, SEMICOLON},
	}

	for _, c := range cases {
		got, err := GetTag(c.Param)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if got.Id != c.Tag.Id {
			t.Errorf("Unexpected tag: got %v, expected: %v", got, c.Tag)
		}
		if got.Id != c.Param {
			t.Errorf("Mismatch tag and parameter: got %v, expected: %v", got, c.Param)
		}
	}
}
