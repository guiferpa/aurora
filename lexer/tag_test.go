package lexer

import (
	"bytes"
	"strings"
	"testing"
)

func TestGetTag(t *testing.T) {
	cases := []struct {
		Tag   Tag
		Param string
	}{
		{TagIdent, IDENT},
		{TagAssign, ASSIGN},
		{TagOParen, O_PAREN},
		{TagCParen, C_PAREN},
		{TagEquals, EQUALS},
		{TagDifferent, DIFFERENT},
		{TagBigger, BIGGER},
		{TagSmaller, SMALLER},
		{TagSum, SUM},
		{TagSub, SUB},
		{TagMult, MULT},
		{TagDiv, DIV},
		{TagComment, COMMENT_LINE},
		{TagOBrk, O_BRK},
		{TagCBrk, C_BRK},
		{TagOCurBrk, O_CUR_BRK},
		{TagCCurBrk, C_CUR_BRK},
		{TagComma, COMMA},
		{TagIf, IF},
		{TagColon, COLON},
		{TagSemicolon, SEMICOLON},
		{TagNumber, NUMBER},
		{TagWhitespace, WHITESPACE},
		{TagBreakLine, BREAK_LINE},
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

func TestMatchTagRule(t *testing.T) {
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
		{[]byte(`0d?`), "", []byte(""), false}, // Exception
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

		// WHITESPACE
		{[]byte(`  `), WHITESPACE, []byte(`  `), true},

		// BREAK_LINE
		{[]byte(`
`), BREAK_LINE, []byte(`
`), true},
	}
	for _, c := range cases {
		matched, tag, match := MatchTagRule(c.Buffer)
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
