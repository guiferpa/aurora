package parser

import (
	"reflect"
	"testing"

	"github.com/guiferpa/aurora/lexer"
)

type tok struct {
	match []byte
	tag   lexer.Tag
}

func (t tok) GetMatch() []byte {
	return t.match
}

func (t tok) GetTag() lexer.Tag {
	return t.tag
}

func (t tok) GetLine() int {
	return 0
}

func (t tok) GetColumn() int {
	return 0
}

func (t tok) GetCursor() int {
	return 0
}

func TestEatTokenWithEmptySlice(t *testing.T) {
	tokens := []lexer.Token{}
	p := &pr{cursor: 0, tokens: tokens}
	got, err := p.EatToken(lexer.BREAK_LINE)
	if err != nil {
		t.Error("unexpected error when eat some token:", err)
	}
	if got != nil {
		t.Errorf("unexpected token when try eat empty slice, got: %v", got)
	}
}

func TestEatTokenWithMismatch(t *testing.T) {
	tokens := []lexer.Token{
		tok{nil, lexer.TagAssign},
		tok{nil, lexer.TagAssign},
		tok{nil, lexer.TagSum},
	}
	p := &pr{cursor: 0, tokens: tokens}
	expected := lexer.IDENT
	got, err := p.EatToken(expected)
	if err == nil {
		t.Error("unexpected error equals nil when eat some token")
	}
	if got.GetTag().Id == expected {
		t.Errorf("unexpected token when try eat, got: %v", got)
	}
}

func TestEatToken(t *testing.T) {
	cases := []struct {
		Tokens   []lexer.Token
		TokenIds []string
	}{
		// = = +
		{
			[]lexer.Token{
				tok{nil, lexer.TagAssign},
				tok{nil, lexer.TagAssign},
				tok{nil, lexer.TagSum},
			},
			[]string{
				lexer.ASSIGN,
				lexer.ASSIGN,
				lexer.SUM,
			},
		},
	}

	for _, c := range cases {
		p := &pr{cursor: 0, tokens: c.Tokens}
		for _, tid := range c.TokenIds {
			got, err := p.EatToken(tid)
			if err != nil {
				t.Error("unexpected error when eat some token:", err)
			}
			if got.GetTag().Id != tid {
				t.Errorf("unexpected token when eat, got: %v, expected: %s", got.GetTag().Id, tid)
			}
		}
	}
}

func TestParse(t *testing.T) {
	// FIX: Improved this test
	t.Skip()

	cases := []struct {
		Tokens   []lexer.Token
		Expected AST
	}{
		// ident a = if 10 bigger 11 { 0; } else { 1; };
		{
			[]lexer.Token{
				tok{[]byte("tok1"), lexer.TagIdent},
				tok{[]byte("a"), lexer.TagId},
				tok{[]byte("tok3"), lexer.TagAssign},
				tok{[]byte("tok4"), lexer.TagIf},
				tok{[]byte("10"), lexer.TagNumber},
				tok{[]byte("tok6"), lexer.TagBigger},
				tok{[]byte("11"), lexer.TagNumber},
				tok{[]byte("tok8"), lexer.TagOCurBrk},
				tok{[]byte("-1"), lexer.TagNumber},
				tok{[]byte("tok10"), lexer.TagSemicolon},
				tok{[]byte("tok11"), lexer.TagCCurBrk},
				tok{[]byte("tok12"), lexer.TagElse},
				tok{[]byte("tok13"), lexer.TagOCurBrk},
				tok{[]byte("1"), lexer.TagNumber},
				tok{[]byte("tok15"), lexer.TagSemicolon},
				tok{[]byte("tok16"), lexer.TagCCurBrk},
				tok{[]byte("tok17"), lexer.TagSemicolon},
				tok{[]byte("tok18"), lexer.TagEOF},
			},
			AST{
				Module: ModuleNode{
					Name: "main",
					Statements: []Node{
						IdentStatementNode{
							Id: "a",
							Expression: IfExpressionNode{
								Test: RelativeExpression{},
								Body: []Node{
									NumberLiteralNode{Value: 0},
								},
								Else: &ElseExpressionNode{
									Body: []Node{
										NumberLiteralNode{Value: 1},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		p := &pr{cursor: 0, tokens: c.Tokens}
		ast, err := p.Parse()
		if err != nil {
			t.Errorf("param: %v, %v", c.Tokens, err)
		}
		if !reflect.DeepEqual(ast, c.Expected) {
			t.Errorf("\nexpected: %v,\ngot: %v", c.Expected, ast)
		}
	}
}
