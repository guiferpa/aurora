package lexer

import (
	"fmt"
	"github.com/guiferpa/aurora/wire/token"
	"reflect"
	"testing"
)

func TestGetTokens(t *testing.T) {
	cases := []struct {
		Buffer []byte
		Tokens []token.Token
	}{
		{
			[]byte(`ident a = 1;`),
			[]token.Token{
				token.New([]byte("ident"), token.TagIdent, 1, 1, 0),
				token.New([]byte(" "), token.TagWhitespace, 1, 6, 5),
				token.New([]byte("a"), token.TagId, 1, 7, 6),
				token.New([]byte(" "), token.TagWhitespace, 1, 8, 7),
				token.New([]byte("="), token.TagAssign, 1, 9, 8),
				token.New([]byte(" "), token.TagWhitespace, 1, 10, 9),
				token.New([]byte("1"), token.TagNumber, 1, 11, 10),
				token.New([]byte(";"), token.TagSemicolon, 1, 12, 11),
				token.New([]byte{}, token.TagEOF, 1, 13, 12),
			},
		},
		{
			[]byte(`ident a = 0xFF;`),
			[]token.Token{
				token.New([]byte("ident"), token.TagIdent, 1, 1, 0),
				token.New([]byte(" "), token.TagWhitespace, 1, 6, 5),
				token.New([]byte("a"), token.TagId, 1, 7, 6),
				token.New([]byte(" "), token.TagWhitespace, 1, 8, 7),
				token.New([]byte("="), token.TagAssign, 1, 9, 8),
				token.New([]byte(" "), token.TagWhitespace, 1, 10, 9),
				token.New([]byte("0xFF"), token.TagNumber, 1, 11, 10),
				token.New([]byte(";"), token.TagSemicolon, 1, 15, 14),
				token.New([]byte{}, token.TagEOF, 1, 16, 15),
			},
		},
		{
			[]byte(`ident hex = 0x10 + 0x20;`),
			[]token.Token{
				token.New([]byte("ident"), token.TagIdent, 1, 1, 0),
				token.New([]byte(" "), token.TagWhitespace, 1, 6, 5),
				token.New([]byte("hex"), token.TagId, 1, 7, 6),
				token.New([]byte(" "), token.TagWhitespace, 1, 10, 9),
				token.New([]byte("="), token.TagAssign, 1, 11, 10),
				token.New([]byte(" "), token.TagWhitespace, 1, 12, 11),
				token.New([]byte("0x10"), token.TagNumber, 1, 13, 12),
				token.New([]byte(" "), token.TagWhitespace, 1, 17, 16),
				token.New([]byte("+"), token.TagSum, 1, 18, 17),
				token.New([]byte(" "), token.TagWhitespace, 1, 19, 18),
				token.New([]byte("0x20"), token.TagNumber, 1, 20, 19),
				token.New([]byte(";"), token.TagSemicolon, 1, 24, 23),
				token.New([]byte{}, token.TagEOF, 1, 25, 24),
			},
		},
		{
			[]byte(`ident tape_hex = [0xFF, 0x10, 0x1A];`),
			[]token.Token{
				token.New([]byte("ident"), token.TagIdent, 1, 1, 0),
				token.New([]byte(" "), token.TagWhitespace, 1, 6, 5),
				token.New([]byte("tape_hex"), token.TagId, 1, 7, 6),
				token.New([]byte(" "), token.TagWhitespace, 1, 15, 14),
				token.New([]byte("="), token.TagAssign, 1, 16, 15),
				token.New([]byte(" "), token.TagWhitespace, 1, 17, 16),
				token.New([]byte("["), token.TagOBrk, 1, 18, 17),
				token.New([]byte("0xFF"), token.TagNumber, 1, 19, 18),
				token.New([]byte(","), token.TagComma, 1, 23, 22),
				token.New([]byte(" "), token.TagWhitespace, 1, 24, 23),
				token.New([]byte("0x10"), token.TagNumber, 1, 25, 24),
				token.New([]byte(","), token.TagComma, 1, 29, 28),
				token.New([]byte(" "), token.TagWhitespace, 1, 30, 29),
				token.New([]byte("0x1A"), token.TagNumber, 1, 31, 30),
				token.New([]byte("]"), token.TagCBrk, 1, 35, 34),
				token.New([]byte(";"), token.TagSemicolon, 1, 36, 35),
				token.New([]byte{}, token.TagEOF, 1, 37, 36),
			},
		},
		{
			[]byte(`ident a = () {
 ident b = 3 + 1_000;
};`),
			[]token.Token{
				token.New([]byte("ident"), token.TagIdent, 1, 1, 0),
				token.New([]byte(" "), token.TagWhitespace, 1, 6, 5),
				token.New([]byte("a"), token.TagId, 1, 7, 6),
				token.New([]byte(" "), token.TagWhitespace, 1, 8, 7),
				token.New([]byte("="), token.TagAssign, 1, 9, 8),
				token.New([]byte(" "), token.TagWhitespace, 1, 10, 9),
				token.New([]byte("("), token.TagOParen, 1, 11, 10),
				token.New([]byte(")"), token.TagCParen, 1, 12, 11),
				token.New([]byte(" "), token.TagWhitespace, 1, 13, 12),
				token.New([]byte("{"), token.TagOCurBrk, 1, 14, 13),
				token.New([]byte(`
`), token.TagBreakLine, 1, 15, 14),
				token.New([]byte(" "), token.TagWhitespace, 2, 1, 15),
				token.New([]byte("ident"), token.TagIdent, 2, 2, 16),
				token.New([]byte(" "), token.TagWhitespace, 2, 7, 21),
				token.New([]byte("b"), token.TagId, 2, 8, 22),
				token.New([]byte(" "), token.TagWhitespace, 2, 9, 23),
				token.New([]byte("="), token.TagAssign, 2, 10, 24),
				token.New([]byte(" "), token.TagWhitespace, 2, 11, 25),
				token.New([]byte("3"), token.TagNumber, 2, 12, 26),
				token.New([]byte(" "), token.TagWhitespace, 2, 13, 27),
				token.New([]byte("+"), token.TagSum, 2, 14, 28),
				token.New([]byte(" "), token.TagWhitespace, 2, 15, 29),
				token.New([]byte("1_000"), token.TagNumber, 2, 16, 30),
				token.New([]byte(";"), token.TagSemicolon, 2, 21, 35),
				token.New([]byte(`
`), token.TagBreakLine, 2, 22, 36),
				token.New([]byte("}"), token.TagCCurBrk, 3, 1, 37),
				token.New([]byte(";"), token.TagSemicolon, 3, 2, 38),
				token.New([]byte{}, token.TagEOF, 3, 3, 39),
			},
		},
		{
			[]byte(`ident a = () {


 ident b = 3 + 1_000;
};`),
			[]token.Token{
				token.New([]byte("ident"), token.TagIdent, 1, 1, 0),
				token.New([]byte(" "), token.TagWhitespace, 1, 6, 5),
				token.New([]byte("a"), token.TagId, 1, 7, 6),
				token.New([]byte(" "), token.TagWhitespace, 1, 8, 7),
				token.New([]byte("="), token.TagAssign, 1, 9, 8),
				token.New([]byte(" "), token.TagWhitespace, 1, 10, 9),
				token.New([]byte("("), token.TagOParen, 1, 11, 10),
				token.New([]byte(")"), token.TagCParen, 1, 12, 11),
				token.New([]byte(" "), token.TagWhitespace, 1, 13, 12),
				token.New([]byte("{"), token.TagOCurBrk, 1, 14, 13),
				token.New([]byte(`
`), token.TagBreakLine, 1, 15, 14),
				token.New([]byte(`
`), token.TagBreakLine, 2, 1, 15),
				token.New([]byte(`
`), token.TagBreakLine, 3, 1, 16),
				token.New([]byte(" "), token.TagWhitespace, 4, 1, 17),
				token.New([]byte("ident"), token.TagIdent, 4, 2, 18),
				token.New([]byte(" "), token.TagWhitespace, 4, 7, 23),
				token.New([]byte("b"), token.TagId, 4, 8, 24),
				token.New([]byte(" "), token.TagWhitespace, 4, 9, 25),
				token.New([]byte("="), token.TagAssign, 4, 10, 26),
				token.New([]byte(" "), token.TagWhitespace, 4, 11, 27),
				token.New([]byte("3"), token.TagNumber, 4, 12, 28),
				token.New([]byte(" "), token.TagWhitespace, 4, 13, 29),
				token.New([]byte("+"), token.TagSum, 4, 14, 30),
				token.New([]byte(" "), token.TagWhitespace, 4, 15, 31),
				token.New([]byte("1_000"), token.TagNumber, 4, 16, 32),
				token.New([]byte(";"), token.TagSemicolon, 4, 21, 37),
				token.New([]byte(`
`), token.TagBreakLine, 4, 22, 38),
				token.New([]byte("}"), token.TagCCurBrk, 5, 1, 39),
				token.New([]byte(";"), token.TagSemicolon, 5, 2, 40),
				token.New([]byte{}, token.TagEOF, 5, 3, 41),
			},
		},
		{
			[]byte(`ident rl = {
  3 + 1_000;
};`),
			[]token.Token{
				token.New([]byte("ident"), token.TagIdent, 1, 1, 0),
				token.New([]byte(" "), token.TagWhitespace, 1, 6, 5),
				token.New([]byte("rl"), token.TagId, 1, 7, 6),
				token.New([]byte(" "), token.TagWhitespace, 1, 9, 8),
				token.New([]byte("="), token.TagAssign, 1, 10, 9),
				token.New([]byte(" "), token.TagWhitespace, 1, 11, 10),
				token.New([]byte("{"), token.TagOCurBrk, 1, 12, 11),
				token.New([]byte(`
`), token.TagBreakLine, 1, 13, 12),
				token.New([]byte("  "), token.TagWhitespace, 2, 1, 13),
				token.New([]byte("3"), token.TagNumber, 2, 3, 15),
				token.New([]byte(" "), token.TagWhitespace, 2, 4, 16),
				token.New([]byte("+"), token.TagSum, 2, 5, 17),
				token.New([]byte(" "), token.TagWhitespace, 2, 6, 18),
				token.New([]byte("1_000"), token.TagNumber, 2, 7, 19),
				token.New([]byte(";"), token.TagSemicolon, 2, 12, 24),
				token.New([]byte(`
`), token.TagBreakLine, 2, 13, 25),
				token.New([]byte("}"), token.TagCCurBrk, 3, 1, 26),
				token.New([]byte(";"), token.TagSemicolon, 3, 2, 27),
				token.New([]byte{}, token.TagEOF, 3, 3, 28),
			},
		},
		{
			[]byte(`ident a = branch {
  op equals 1: sum(1, 1),
  10;
};`),
			[]token.Token{
				token.New([]byte("ident"), token.TagIdent, 1, 1, 0),
				token.New([]byte(" "), token.TagWhitespace, 1, 6, 5),
				token.New([]byte("a"), token.TagId, 1, 7, 6),
				token.New([]byte(" "), token.TagWhitespace, 1, 8, 7),
				token.New([]byte("="), token.TagAssign, 1, 9, 8),
				token.New([]byte(" "), token.TagWhitespace, 1, 10, 9),
				token.New([]byte("branch"), token.TagBranch, 1, 11, 10),
				token.New([]byte(" "), token.TagWhitespace, 1, 17, 16),
				token.New([]byte("{"), token.TagOCurBrk, 1, 18, 17),
				token.New([]byte(`
`), token.TagBreakLine, 1, 19, 18),
				token.New([]byte("  "), token.TagWhitespace, 2, 1, 19),
				token.New([]byte("op"), token.TagId, 2, 3, 21),
				token.New([]byte(" "), token.TagWhitespace, 2, 5, 23),
				token.New([]byte("equals"), token.TagEquals, 2, 6, 24),
				token.New([]byte(" "), token.TagWhitespace, 2, 12, 30),
				token.New([]byte("1"), token.TagNumber, 2, 13, 31),
				token.New([]byte(":"), token.TagColon, 2, 14, 32),
				token.New([]byte(" "), token.TagWhitespace, 2, 15, 33),
				token.New([]byte("sum"), token.TagId, 2, 16, 34),
				token.New([]byte("("), token.TagOParen, 2, 19, 37),
				token.New([]byte("1"), token.TagNumber, 2, 20, 38),
				token.New([]byte(","), token.TagComma, 2, 21, 39),
				token.New([]byte(" "), token.TagWhitespace, 2, 22, 40),
				token.New([]byte("1"), token.TagNumber, 2, 23, 41),
				token.New([]byte(")"), token.TagCParen, 2, 24, 42),
				token.New([]byte(","), token.TagComma, 2, 25, 43),
				token.New([]byte(`
`), token.TagBreakLine, 2, 26, 44),
				token.New([]byte("  "), token.TagWhitespace, 3, 1, 45),
				token.New([]byte("10"), token.TagNumber, 3, 3, 47),
				token.New([]byte(";"), token.TagSemicolon, 3, 5, 49),
				token.New([]byte(`
`), token.TagBreakLine, 3, 6, 50),
				token.New([]byte("}"), token.TagCCurBrk, 4, 1, 51),
				token.New([]byte(";"), token.TagSemicolon, 4, 2, 52),
				token.New([]byte{}, token.TagEOF, 4, 3, 53),
			},
		},
	}
	for _, c := range cases {
		tokens, err := New(NewLexerOptions{}).GetTokens(c.Buffer)
		if err != nil {
			t.Errorf("param: %v, %v", string(c.Buffer), err)
		}
		if !reflect.DeepEqual(tokens, c.Tokens) {
			for i, v := range tokens {
				// Improve log for testing
				tok := c.Tokens[i]
				fmt.Println(v.GetLine(), v.GetColumn(), v.GetCursor(), v.GetTag().Id, string(v.GetMatch()), "<==>", tok.GetLine(), tok.GetColumn(), tok.GetCursor(), tok.GetTag().Id, string(tok.GetMatch()))
			}
			t.Errorf("\nexpected: %v,\ngot: %v", c.Tokens, tokens)
		}
	}
}
