package parser

import (
	"testing"

	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/token"
)

type tok struct {
	match []byte
	tag   token.Tag
}

func (t tok) GetMatch() []byte {
	return t.match
}

func (t tok) GetTag() token.Tag {
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
	tokens := []token.Token{}
	p := &pr{cursor: 0, tokens: tokens}
	got, err := p.EatToken(token.BREAK_LINE)
	if err != nil {
		t.Error("unexpected error when eat some token:", err)
	}
	if got != nil {
		t.Errorf("unexpected token when try eat empty slice, got: %v", got)
	}
}

func TestEatTokenWithMismatch(t *testing.T) {
	tokens := []token.Token{
		tok{nil, token.TagAssign},
		tok{nil, token.TagAssign},
		tok{nil, token.TagSum},
	}
	p := New(NewParserOptions{
		Tokens: tokens,
	})
	expected := token.IDENT
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
		Tokens   []token.Token
		TokenIds []string
	}{
		// = = +
		{
			[]token.Token{
				tok{nil, token.TagAssign},
				tok{nil, token.TagAssign},
				tok{nil, token.TagSum},
			},
			[]string{
				token.ASSIGN,
				token.ASSIGN,
				token.SUM,
			},
		},
	}

	for _, c := range cases {
		p := New(NewParserOptions{
			Tokens: c.Tokens,
		})
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
	// ident a = if 10 bigger 11 { 0; } else { 1; };
	tokens := []token.Token{
		tok{[]byte("tok1"), token.TagIdent},
		tok{[]byte("a"), token.TagId},
		tok{[]byte("tok3"), token.TagAssign},
		tok{[]byte("tok4"), token.TagIf},
		tok{[]byte("10"), token.TagNumber},
		tok{[]byte("tok6"), token.TagBigger},
		tok{[]byte("11"), token.TagNumber},
		tok{[]byte("tok8"), token.TagOCurBrk},
		tok{[]byte("0"), token.TagNumber},
		tok{[]byte("tok10"), token.TagSemicolon},
		tok{[]byte("tok11"), token.TagCCurBrk},
		tok{[]byte("tok12"), token.TagElse},
		tok{[]byte("tok13"), token.TagOCurBrk},
		tok{[]byte("1"), token.TagNumber},
		tok{[]byte("tok15"), token.TagSemicolon},
		tok{[]byte("tok16"), token.TagCCurBrk},
		tok{[]byte("tok17"), token.TagSemicolon},
		tok{[]byte("tok18"), token.TagEOF},
	}
	expected := ast.AST{
		Filename: "testing.ar",
		Nodes: []ast.Node{
			ast.IdentLiteral{
				Id:    "a",
				Token: tokens[1],
				Value: ast.IfExpression{
					Test: ast.RelativeExpression{
						Left:      ast.NumberLiteral{Value: 10, Token: tokens[4]},
						Right:     ast.NumberLiteral{Value: 11, Token: tokens[6]},
						Operation: ast.OperationLiteral{Value: "tok6", Token: tokens[5]},
					},
					Body: []ast.Node{
						ast.NumberLiteral{Value: 0, Token: tokens[8]},
					},
					Else: &ast.ElseExpression{
						Body: []ast.Node{
							ast.NumberLiteral{Value: 1, Token: tokens[13]},
						},
					},
				},
			},
		},
	}
	p := &pr{filename: "testing.ar", cursor: 0, tokens: tokens}
	tree, err := p.Parse()
	if err != nil {
		t.Errorf("param: %v, %v", tokens, err)
	}
	if !ast.Equal(tree, expected) {
		t.Errorf("\nexpected: %+v,\ngot: %+v", expected, tree)
	}
}
