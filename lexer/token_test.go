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
			[]byte(`ident a = 0xFF;`),
			[]Token{
				tok{1, 1, 0, TagIdent, []byte("ident")},
				tok{1, 6, 5, TagWhitespace, []byte(" ")},
				tok{1, 7, 6, TagId, []byte("a")},
				tok{1, 8, 7, TagWhitespace, []byte(" ")},
				tok{1, 9, 8, TagAssign, []byte("=")},
				tok{1, 10, 9, TagWhitespace, []byte(" ")},
				tok{1, 11, 10, TagNumber, []byte("0xFF")},
				tok{1, 15, 14, TagSemicolon, []byte(";")},
				tok{1, 16, 15, TagEOF, []byte{}},
			},
		},
		{
			[]byte(`ident hex = 0x10 + 0x20;`),
			[]Token{
				tok{1, 1, 0, TagIdent, []byte("ident")},
				tok{1, 6, 5, TagWhitespace, []byte(" ")},
				tok{1, 7, 6, TagId, []byte("hex")},
				tok{1, 10, 9, TagWhitespace, []byte(" ")},
				tok{1, 11, 10, TagAssign, []byte("=")},
				tok{1, 12, 11, TagWhitespace, []byte(" ")},
				tok{1, 13, 12, TagNumber, []byte("0x10")},
				tok{1, 17, 16, TagWhitespace, []byte(" ")},
				tok{1, 18, 17, TagSum, []byte("+")},
				tok{1, 19, 18, TagWhitespace, []byte(" ")},
				tok{1, 20, 19, TagNumber, []byte("0x20")},
				tok{1, 24, 23, TagSemicolon, []byte(";")},
				tok{1, 25, 24, TagEOF, []byte{}},
			},
		},
		{
			[]byte(`ident tape_hex = [0xFF, 0x10, 0x1A];`),
			[]Token{
				tok{1, 1, 0, TagIdent, []byte("ident")},
				tok{1, 6, 5, TagWhitespace, []byte(" ")},
				tok{1, 7, 6, TagId, []byte("tape_hex")},
				tok{1, 15, 14, TagWhitespace, []byte(" ")},
				tok{1, 16, 15, TagAssign, []byte("=")},
				tok{1, 17, 16, TagWhitespace, []byte(" ")},
				tok{1, 18, 17, TagOBrk, []byte("[")},
				tok{1, 19, 18, TagNumber, []byte("0xFF")},
				tok{1, 23, 22, TagComma, []byte(",")},
				tok{1, 24, 23, TagWhitespace, []byte(" ")},
				tok{1, 25, 24, TagNumber, []byte("0x10")},
				tok{1, 29, 28, TagComma, []byte(",")},
				tok{1, 30, 29, TagWhitespace, []byte(" ")},
				tok{1, 31, 30, TagNumber, []byte("0x1A")},
				tok{1, 35, 34, TagCBrk, []byte("]")},
				tok{1, 36, 35, TagSemicolon, []byte(";")},
				tok{1, 37, 36, TagEOF, []byte{}},
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
		{
			[]byte(`ident a = branch {
  op equals 1: sum(1, 1),
  10;
};`),
			[]Token{
				tok{1, 1, 0, TagIdent, []byte("ident")},
				tok{1, 6, 5, TagWhitespace, []byte(" ")},
				tok{1, 7, 6, TagId, []byte("a")},
				tok{1, 8, 7, TagWhitespace, []byte(" ")},
				tok{1, 9, 8, TagAssign, []byte("=")},
				tok{1, 10, 9, TagWhitespace, []byte(" ")},
				tok{1, 11, 10, TagBranch, []byte("branch")},
				tok{1, 17, 16, TagWhitespace, []byte(" ")},
				tok{1, 18, 17, TagOCurBrk, []byte("{")},
				tok{1, 19, 18, TagBreakLine, []byte(`
`)},
				tok{2, 1, 19, TagWhitespace, []byte("  ")},
				tok{2, 3, 21, TagId, []byte("op")},
				tok{2, 5, 23, TagWhitespace, []byte(" ")},
				tok{2, 6, 24, TagEquals, []byte("equals")},
				tok{2, 12, 30, TagWhitespace, []byte(" ")},
				tok{2, 13, 31, TagNumber, []byte("1")},
				tok{2, 14, 32, TagColon, []byte(":")},
				tok{2, 15, 33, TagWhitespace, []byte(" ")},
				tok{2, 16, 34, TagId, []byte("sum")},
				tok{2, 19, 37, TagOParen, []byte("(")},
				tok{2, 20, 38, TagNumber, []byte("1")},
				tok{2, 21, 39, TagComma, []byte(",")},
				tok{2, 22, 40, TagWhitespace, []byte(" ")},
				tok{2, 23, 41, TagNumber, []byte("1")},
				tok{2, 24, 42, TagCParen, []byte(")")},
				tok{2, 25, 43, TagComma, []byte(",")},
				tok{2, 26, 44, TagBreakLine, []byte(`
`)},
				tok{3, 1, 45, TagWhitespace, []byte("  ")},
				tok{3, 3, 47, TagNumber, []byte("10")},
				tok{3, 5, 49, TagSemicolon, []byte(";")},
				tok{3, 6, 50, TagBreakLine, []byte(`
`)},
				tok{4, 1, 51, TagCCurBrk, []byte("}")},
				tok{4, 2, 52, TagSemicolon, []byte(";")},
				tok{4, 3, 53, TagEOF, []byte{}},
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
