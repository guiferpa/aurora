package lexer

import (
	"fmt"
	"reflect"
	"testing"
)

func TestGetTokens(t *testing.T) {
	cases := []struct {
		Buffer []byte
		Tokens []Token
	}{
		{
			[]byte(`ident a = 1;`),
			[]Token{
				tok{1, 1, 0, TagIdent, []byte("ident")},
				tok{1, 6, 5, TagWhitespace, []byte(" ")},
				tok{1, 7, 6, TagId, []byte("a")},
				tok{1, 8, 7, TagWhitespace, []byte(" ")},
				tok{1, 9, 8, TagAssign, []byte("=")},
				tok{1, 10, 9, TagWhitespace, []byte(" ")},
				tok{1, 11, 10, TagNumber, []byte("1")},
				tok{1, 12, 11, TagSemicolon, []byte(";")},
				tok{1, 13, 12, TagEOF, []byte{}},
			},
		},
		{
			[]byte(`ident a = () {
 ident b = 3 + 1_000;
};`),
			[]Token{
				tok{1, 1, 0, TagIdent, []byte("ident")},
				tok{1, 6, 5, TagWhitespace, []byte(" ")},
				tok{1, 7, 6, TagId, []byte("a")},
				tok{1, 8, 7, TagWhitespace, []byte(" ")},
				tok{1, 9, 8, TagAssign, []byte("=")},
				tok{1, 10, 9, TagWhitespace, []byte(" ")},
				tok{1, 11, 10, TagOParen, []byte("(")},
				tok{1, 12, 11, TagCParen, []byte(")")},
				tok{1, 13, 12, TagWhitespace, []byte(" ")},
				tok{1, 14, 13, TagOCurBrk, []byte("{")},
				tok{1, 15, 14, TagBreakLine, []byte(`
`)},
				tok{2, 1, 15, TagWhitespace, []byte(" ")},
				tok{2, 2, 16, TagIdent, []byte("ident")},
				tok{2, 7, 21, TagWhitespace, []byte(" ")},
				tok{2, 8, 22, TagId, []byte("b")},
				tok{2, 9, 23, TagWhitespace, []byte(" ")},
				tok{2, 10, 24, TagAssign, []byte("=")},
				tok{2, 11, 25, TagWhitespace, []byte(" ")},
				tok{2, 12, 26, TagNumber, []byte("3")},
				tok{2, 13, 27, TagWhitespace, []byte(" ")},
				tok{2, 14, 28, TagSum, []byte("+")},
				tok{2, 15, 29, TagWhitespace, []byte(" ")},
				tok{2, 16, 30, TagNumber, []byte("1_000")},
				tok{2, 21, 35, TagSemicolon, []byte(";")},
				tok{2, 22, 36, TagBreakLine, []byte(`
`)},
				tok{3, 1, 37, TagCCurBrk, []byte("}")},
				tok{3, 2, 38, TagSemicolon, []byte(";")},
				tok{3, 3, 39, TagEOF, []byte{}},
			},
		},
		{
			[]byte(`ident a = () {


 ident b = 3 + 1_000;
};`),
			[]Token{
				tok{1, 1, 0, TagIdent, []byte("ident")},
				tok{1, 6, 5, TagWhitespace, []byte(" ")},
				tok{1, 7, 6, TagId, []byte("a")},
				tok{1, 8, 7, TagWhitespace, []byte(" ")},
				tok{1, 9, 8, TagAssign, []byte("=")},
				tok{1, 10, 9, TagWhitespace, []byte(" ")},
				tok{1, 11, 10, TagOParen, []byte("(")},
				tok{1, 12, 11, TagCParen, []byte(")")},
				tok{1, 13, 12, TagWhitespace, []byte(" ")},
				tok{1, 14, 13, TagOCurBrk, []byte("{")},
				tok{1, 15, 14, TagBreakLine, []byte(`
`)},
				tok{2, 1, 15, TagBreakLine, []byte(`
`)},
				tok{3, 1, 16, TagBreakLine, []byte(`
`)},
				tok{4, 1, 17, TagWhitespace, []byte(" ")},
				tok{4, 2, 18, TagIdent, []byte("ident")},
				tok{4, 7, 23, TagWhitespace, []byte(" ")},
				tok{4, 8, 24, TagId, []byte("b")},
				tok{4, 9, 25, TagWhitespace, []byte(" ")},
				tok{4, 10, 26, TagAssign, []byte("=")},
				tok{4, 11, 27, TagWhitespace, []byte(" ")},
				tok{4, 12, 28, TagNumber, []byte("3")},
				tok{4, 13, 29, TagWhitespace, []byte(" ")},
				tok{4, 14, 30, TagSum, []byte("+")},
				tok{4, 15, 31, TagWhitespace, []byte(" ")},
				tok{4, 16, 32, TagNumber, []byte("1_000")},
				tok{4, 21, 37, TagSemicolon, []byte(";")},
				tok{4, 22, 38, TagBreakLine, []byte(`
`)},
				tok{5, 1, 39, TagCCurBrk, []byte("}")},
				tok{5, 2, 40, TagSemicolon, []byte(";")},
				tok{5, 3, 41, TagEOF, []byte{}},
			},
		},
		{
			[]byte(`ident rl = {
  3 + 1_000;
};`),
			[]Token{
				tok{1, 1, 0, TagIdent, []byte("ident")},
				tok{1, 6, 5, TagWhitespace, []byte(" ")},
				tok{1, 7, 6, TagId, []byte("rl")},
				tok{1, 9, 8, TagWhitespace, []byte(" ")},
				tok{1, 10, 9, TagAssign, []byte("=")},
				tok{1, 11, 10, TagWhitespace, []byte(" ")},
				tok{1, 12, 11, TagOCurBrk, []byte("{")},
				tok{1, 13, 12, TagBreakLine, []byte(`
`)},
				tok{2, 1, 13, TagWhitespace, []byte("  ")},
				tok{2, 3, 15, TagNumber, []byte("3")},
				tok{2, 4, 16, TagWhitespace, []byte(" ")},
				tok{2, 5, 17, TagSum, []byte("+")},
				tok{2, 6, 18, TagWhitespace, []byte(" ")},
				tok{2, 7, 19, TagNumber, []byte("1_000")},
				tok{2, 12, 24, TagSemicolon, []byte(";")},
				tok{2, 13, 25, TagBreakLine, []byte(`
`)},
				tok{3, 1, 26, TagCCurBrk, []byte("}")},
				tok{3, 2, 27, TagSemicolon, []byte(";")},
				tok{3, 3, 28, TagEOF, []byte{}},
			},
		},
	}
	for _, c := range cases {
		tokens, err := GetTokens(c.Buffer)
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
