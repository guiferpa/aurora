package lexer

import (
	"fmt"
	"reflect"
	"testing"
)

func TestGetTokensGivenBytes(t *testing.T) {
	cases := []struct {
		Buffer []byte
		Tokens []Token
	}{
		{
			[]byte(`ident a = 1;`),
			[]Token{
				tok{1, 1, tIdent, []byte("ident")},
				tok{1, 6, tWhitespace, []byte(" ")},
				tok{1, 7, tId, []byte("a")},
				tok{1, 8, tWhitespace, []byte(" ")},
				tok{1, 9, tAssign, []byte("=")},
				tok{1, 10, tWhitespace, []byte(" ")},
				tok{1, 11, tNumber, []byte("1")},
				tok{1, 12, tSemicolon, []byte(";")},
				tok{1, 13, tEndOfBuffer, []byte{}},
			},
		},
		{
			[]byte(`ident a = () {
 ident b = 3 + 1_000;
};`),
			[]Token{
				tok{1, 1, tIdent, []byte("ident")},
				tok{1, 6, tWhitespace, []byte(" ")},
				tok{1, 7, tId, []byte("a")},
				tok{1, 8, tWhitespace, []byte(" ")},
				tok{1, 9, tAssign, []byte("=")},
				tok{1, 10, tWhitespace, []byte(" ")},
				tok{1, 11, tOParen, []byte("(")},
				tok{1, 12, tCParen, []byte(")")},
				tok{1, 13, tWhitespace, []byte(" ")},
				tok{1, 14, tOCurBrk, []byte("{")},
				tok{1, 15, tBreakLine, []byte(`
`)},
				tok{2, 1, tWhitespace, []byte(" ")},
				tok{2, 2, tIdent, []byte("ident")},
				tok{2, 7, tWhitespace, []byte(" ")},
				tok{2, 8, tId, []byte("b")},
				tok{2, 9, tWhitespace, []byte(" ")},
				tok{2, 10, tAssign, []byte("=")},
				tok{2, 11, tWhitespace, []byte(" ")},
				tok{2, 12, tNumber, []byte("3")},
				tok{2, 13, tWhitespace, []byte(" ")},
				tok{2, 14, tSum, []byte("+")},
				tok{2, 15, tWhitespace, []byte(" ")},
				tok{2, 16, tNumber, []byte("1_000")},
				tok{2, 21, tSemicolon, []byte(";")},
				tok{2, 22, tBreakLine, []byte(`
`)},
				tok{3, 1, tCCurBrk, []byte("}")},
				tok{3, 2, tSemicolon, []byte(";")},
				tok{3, 3, tEndOfBuffer, []byte{}},
			},
		},
		{
			[]byte(`ident a = () {


 ident b = 3 + 1_000;
};`),
			[]Token{
				tok{1, 1, tIdent, []byte("ident")},
				tok{1, 6, tWhitespace, []byte(" ")},
				tok{1, 7, tId, []byte("a")},
				tok{1, 8, tWhitespace, []byte(" ")},
				tok{1, 9, tAssign, []byte("=")},
				tok{1, 10, tWhitespace, []byte(" ")},
				tok{1, 11, tOParen, []byte("(")},
				tok{1, 12, tCParen, []byte(")")},
				tok{1, 13, tWhitespace, []byte(" ")},
				tok{1, 14, tOCurBrk, []byte("{")},
				tok{1, 15, tBreakLine, []byte(`
`)},
				tok{2, 1, tBreakLine, []byte(`
`)},
				tok{3, 1, tBreakLine, []byte(`
`)},
				tok{4, 1, tWhitespace, []byte(" ")},
				tok{4, 2, tIdent, []byte("ident")},
				tok{4, 7, tWhitespace, []byte(" ")},
				tok{4, 8, tId, []byte("b")},
				tok{4, 9, tWhitespace, []byte(" ")},
				tok{4, 10, tAssign, []byte("=")},
				tok{4, 11, tWhitespace, []byte(" ")},
				tok{4, 12, tNumber, []byte("3")},
				tok{4, 13, tWhitespace, []byte(" ")},
				tok{4, 14, tSum, []byte("+")},
				tok{4, 15, tWhitespace, []byte(" ")},
				tok{4, 16, tNumber, []byte("1_000")},
				tok{4, 21, tSemicolon, []byte(";")},
				tok{4, 22, tBreakLine, []byte(`
`)},
				tok{5, 1, tCCurBrk, []byte("}")},
				tok{5, 2, tSemicolon, []byte(";")},
				tok{5, 3, tEndOfBuffer, []byte{}},
			},
		},
		{
			[]byte(`ident rl = {
  3 + 1_000;
};`),
			[]Token{
				tok{1, 1, tIdent, []byte("ident")},
				tok{1, 6, tWhitespace, []byte(" ")},
				tok{1, 7, tId, []byte("rl")},
				tok{1, 9, tWhitespace, []byte(" ")},
				tok{1, 10, tAssign, []byte("=")},
				tok{1, 11, tWhitespace, []byte(" ")},
				tok{1, 12, tOCurBrk, []byte("{")},
				tok{1, 13, tBreakLine, []byte(`
`)},
				tok{2, 1, tWhitespace, []byte("  ")},
				tok{2, 3, tNumber, []byte("3")},
				tok{2, 4, tWhitespace, []byte(" ")},
				tok{2, 5, tSum, []byte("+")},
				tok{2, 6, tWhitespace, []byte(" ")},
				tok{2, 7, tNumber, []byte("1_000")},
				tok{2, 12, tSemicolon, []byte(";")},
				tok{2, 13, tBreakLine, []byte(`
`)},
				tok{3, 1, tCCurBrk, []byte("}")},
				tok{3, 2, tSemicolon, []byte(";")},
				tok{3, 3, tEndOfBuffer, []byte{}},
			},
		},
	}
	for _, c := range cases {
		tokens, err := GetTokensGivenBytes(c.Buffer)
		if err != nil {
			t.Errorf("param: %v, %v", string(c.Buffer), err)
		}
		if !reflect.DeepEqual(tokens, c.Tokens) {
			for i, v := range tokens {
				// Improve log for testing
				fmt.Println(v.GetLine(), v.GetColumn(), v.GetTag().Id, v.GetMatch(), "<==>", c.Tokens[i].GetLine(), c.Tokens[i].GetColumn(), c.Tokens[i].GetTag().Id, c.Tokens[i].GetMatch())
			}
			t.Errorf("\nexpected: %v,\ngot: %v", c.Tokens, tokens)
		}
	}
}
