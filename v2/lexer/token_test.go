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
				tok{1, 1, 0, tIdent, []byte("ident")},
				tok{1, 6, 5, tWhitespace, []byte(" ")},
				tok{1, 7, 6, tId, []byte("a")},
				tok{1, 8, 7, tWhitespace, []byte(" ")},
				tok{1, 9, 8, tAssign, []byte("=")},
				tok{1, 10, 9, tWhitespace, []byte(" ")},
				tok{1, 11, 10, tNumber, []byte("1")},
				tok{1, 12, 11, tSemicolon, []byte(";")},
				tok{1, 13, 12, tEndOfBuffer, []byte{}},
			},
		},
		{
			[]byte(`ident a = () {
 ident b = 3 + 1_000;
};`),
			[]Token{
				tok{1, 1, 0, tIdent, []byte("ident")},
				tok{1, 6, 5, tWhitespace, []byte(" ")},
				tok{1, 7, 6, tId, []byte("a")},
				tok{1, 8, 7, tWhitespace, []byte(" ")},
				tok{1, 9, 8, tAssign, []byte("=")},
				tok{1, 10, 9, tWhitespace, []byte(" ")},
				tok{1, 11, 10, tOParen, []byte("(")},
				tok{1, 12, 11, tCParen, []byte(")")},
				tok{1, 13, 12, tWhitespace, []byte(" ")},
				tok{1, 14, 13, tOCurBrk, []byte("{")},
				tok{1, 15, 14, tBreakLine, []byte(`
`)},
				tok{2, 1, 15, tWhitespace, []byte(" ")},
				tok{2, 2, 16, tIdent, []byte("ident")},
				tok{2, 7, 21, tWhitespace, []byte(" ")},
				tok{2, 8, 22, tId, []byte("b")},
				tok{2, 9, 23, tWhitespace, []byte(" ")},
				tok{2, 10, 24, tAssign, []byte("=")},
				tok{2, 11, 25, tWhitespace, []byte(" ")},
				tok{2, 12, 26, tNumber, []byte("3")},
				tok{2, 13, 27, tWhitespace, []byte(" ")},
				tok{2, 14, 28, tSum, []byte("+")},
				tok{2, 15, 29, tWhitespace, []byte(" ")},
				tok{2, 16, 30, tNumber, []byte("1_000")},
				tok{2, 21, 35, tSemicolon, []byte(";")},
				tok{2, 22, 36, tBreakLine, []byte(`
`)},
				tok{3, 1, 37, tCCurBrk, []byte("}")},
				tok{3, 2, 38, tSemicolon, []byte(";")},
				tok{3, 3, 39, tEndOfBuffer, []byte{}},
			},
		},
		{
			[]byte(`ident a = () {


 ident b = 3 + 1_000;
};`),
			[]Token{
				tok{1, 1, 0, tIdent, []byte("ident")},
				tok{1, 6, 5, tWhitespace, []byte(" ")},
				tok{1, 7, 6, tId, []byte("a")},
				tok{1, 8, 7, tWhitespace, []byte(" ")},
				tok{1, 9, 8, tAssign, []byte("=")},
				tok{1, 10, 9, tWhitespace, []byte(" ")},
				tok{1, 11, 10, tOParen, []byte("(")},
				tok{1, 12, 11, tCParen, []byte(")")},
				tok{1, 13, 12, tWhitespace, []byte(" ")},
				tok{1, 14, 13, tOCurBrk, []byte("{")},
				tok{1, 15, 14, tBreakLine, []byte(`
`)},
				tok{2, 1, 15, tBreakLine, []byte(`
`)},
				tok{3, 1, 16, tBreakLine, []byte(`
`)},
				tok{4, 1, 17, tWhitespace, []byte(" ")},
				tok{4, 2, 18, tIdent, []byte("ident")},
				tok{4, 7, 23, tWhitespace, []byte(" ")},
				tok{4, 8, 24, tId, []byte("b")},
				tok{4, 9, 25, tWhitespace, []byte(" ")},
				tok{4, 10, 26, tAssign, []byte("=")},
				tok{4, 11, 27, tWhitespace, []byte(" ")},
				tok{4, 12, 28, tNumber, []byte("3")},
				tok{4, 13, 29, tWhitespace, []byte(" ")},
				tok{4, 14, 30, tSum, []byte("+")},
				tok{4, 15, 31, tWhitespace, []byte(" ")},
				tok{4, 16, 32, tNumber, []byte("1_000")},
				tok{4, 21, 37, tSemicolon, []byte(";")},
				tok{4, 22, 38, tBreakLine, []byte(`
`)},
				tok{5, 1, 39, tCCurBrk, []byte("}")},
				tok{5, 2, 40, tSemicolon, []byte(";")},
				tok{5, 3, 41, tEndOfBuffer, []byte{}},
			},
		},
		{
			[]byte(`ident rl = {
  3 + 1_000;
};`),
			[]Token{
				tok{1, 1, 0, tIdent, []byte("ident")},
				tok{1, 6, 5, tWhitespace, []byte(" ")},
				tok{1, 7, 6, tId, []byte("rl")},
				tok{1, 9, 8, tWhitespace, []byte(" ")},
				tok{1, 10, 9, tAssign, []byte("=")},
				tok{1, 11, 10, tWhitespace, []byte(" ")},
				tok{1, 12, 11, tOCurBrk, []byte("{")},
				tok{1, 13, 12, tBreakLine, []byte(`
`)},
				tok{2, 1, 13, tWhitespace, []byte("  ")},
				tok{2, 3, 15, tNumber, []byte("3")},
				tok{2, 4, 16, tWhitespace, []byte(" ")},
				tok{2, 5, 17, tSum, []byte("+")},
				tok{2, 6, 18, tWhitespace, []byte(" ")},
				tok{2, 7, 19, tNumber, []byte("1_000")},
				tok{2, 12, 24, tSemicolon, []byte(";")},
				tok{2, 13, 25, tBreakLine, []byte(`
`)},
				tok{3, 1, 26, tCCurBrk, []byte("}")},
				tok{3, 2, 27, tSemicolon, []byte(";")},
				tok{3, 3, 28, tEndOfBuffer, []byte{}},
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
        tok := c.Tokens[i]
				fmt.Println(v.GetLine(), v.GetColumn(), v.GetCursor(), v.GetTag().Id, v.GetMatch(), "<==>", tok.GetLine(), tok.GetColumn(), tok.GetCursor(), tok.GetTag().Id, tok.GetMatch())
			}
			t.Errorf("\nexpected: %v,\ngot: %v", c.Tokens, tokens)
		}
	}
}
