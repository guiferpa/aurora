package parser

import (
	"reflect"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
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
	// ident a = if 10 bigger 11 { 0; } else { 1; };
	tokens := []lexer.Token{
		tok{[]byte("tok1"), lexer.TagIdent},
		tok{[]byte("a"), lexer.TagId},
		tok{[]byte("tok3"), lexer.TagAssign},
		tok{[]byte("tok4"), lexer.TagIf},
		tok{[]byte("10"), lexer.TagNumber},
		tok{[]byte("tok6"), lexer.TagBigger},
		tok{[]byte("11"), lexer.TagNumber},
		tok{[]byte("tok8"), lexer.TagOCurBrk},
		tok{[]byte("0"), lexer.TagNumber},
		tok{[]byte("tok10"), lexer.TagSemicolon},
		tok{[]byte("tok11"), lexer.TagCCurBrk},
		tok{[]byte("tok12"), lexer.TagElse},
		tok{[]byte("tok13"), lexer.TagOCurBrk},
		tok{[]byte("1"), lexer.TagNumber},
		tok{[]byte("tok15"), lexer.TagSemicolon},
		tok{[]byte("tok16"), lexer.TagCCurBrk},
		tok{[]byte("tok17"), lexer.TagSemicolon},
		tok{[]byte("tok18"), lexer.TagEOF},
	}
	expected := AST{
		Module: Module{
			Name: "main",
			Expressions: []Node{
				IdentLiteral{
					Id:    "a",
					Token: tokens[1],
					Value: IfExpression{
						Test: RelativeExpression{
							Left:      NumberLiteral{Value: 10, Token: tokens[4]},
							Right:     NumberLiteral{Value: 11, Token: tokens[6]},
							Operation: OperationLiteral{Value: "tok6", Token: tokens[5]},
						},
						Body: []Node{
							NumberLiteral{Value: 0, Token: tokens[8]},
						},
						Else: &ElseExpression{
							Body: []Node{
								NumberLiteral{Value: 1, Token: tokens[13]},
							},
						},
					},
				},
			},
		},
	}

	p := &pr{cursor: 0, tokens: tokens}
	ast, err := p.Parse()
	if err != nil {
		t.Errorf("param: %v, %v", tokens, err)
	}
	if !reflect.DeepEqual(ast, expected) {
		t.Errorf("\nexpected: %v,\ngot: %v", expected, ast)
	}
}

func TestParseNothing(t *testing.T) {
	semicolon := tok{[]byte(";"), lexer.TagSemicolon}
	eof := tok{[]byte(""), lexer.TagEOF}
	nothing := tok{[]byte("nothing"), lexer.TagNothing}
	sum := tok{[]byte("+"), lexer.TagSum}
	sub := tok{[]byte("-"), lexer.TagSub}
	equals := tok{[]byte("equals"), lexer.TagEquals}
	or := tok{[]byte("or"), lexer.TagOr}
	zero := tok{[]byte("0"), lexer.TagNumber}
	one := tok{[]byte("1"), lexer.TagNumber}
	two := tok{[]byte("2"), lexer.TagNumber}
	three := tok{[]byte("3"), lexer.TagNumber}

	cases := []struct {
		name   string
		tokens []lexer.Token
		want   *Module
	}{
		{
			name:   "top_level",
			tokens: []lexer.Token{nothing, semicolon, eof},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					NothingLiteral{Token: nothing},
				},
			},
		},
		{
			name: "inside_block",
			tokens: []lexer.Token{
				tok{[]byte("{"), lexer.TagOCurBrk}, nothing, semicolon,
				tok{[]byte("}"), lexer.TagCCurBrk}, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					BlockExpression{
						Body: []Node{
							NothingLiteral{Token: nothing},
						},
					},
				},
			},
		},
		{
			name: "rhs_of_assignment",
			tokens: []lexer.Token{
				tok{[]byte("ident"), lexer.TagIdent}, tok{[]byte("x"), lexer.TagId},
				tok{[]byte("="), lexer.TagAssign}, nothing, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					IdentLiteral{
						Id:    "x",
						Token: tok{[]byte("x"), lexer.TagId},
						Value: NothingLiteral{Token: nothing},
					},
				},
			},
		},
		{
			name: "parenthesized",
			tokens: []lexer.Token{
				tok{[]byte("("), lexer.TagOParen}, nothing, tok{[]byte(")"), lexer.TagCParen}, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					NothingLiteral{Token: nothing},
				},
			},
		},
		{
			name: "binary_left_nothing_plus_one",
			tokens: []lexer.Token{
				nothing, sum, one, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					BinaryExpression{
						Left:      NothingLiteral{Token: nothing},
						Right:     NumberLiteral{Value: 1, Token: one},
						Operation: OperationLiteral{Value: "+", Token: sum},
					},
				},
			},
		},
		{
			name: "binary_right_one_plus_nothing",
			tokens: []lexer.Token{
				tok{[]byte("1"), lexer.TagNumber}, sum, nothing, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					BinaryExpression{
						Left:      NumberLiteral{Value: 1, Token: one},
						Right:     NothingLiteral{Token: nothing},
						Operation: OperationLiteral{Value: "+", Token: sum},
					},
				},
			},
		},
		{
			name: "comparison_nothing_equals_zero",
			tokens: []lexer.Token{
				nothing, equals, zero, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					RelativeExpression{
						Left:      NothingLiteral{Token: nothing},
						Right:     NumberLiteral{Value: 0, Token: zero},
						Operation: OperationLiteral{Value: "equals", Token: equals},
					},
				},
			},
		},
		{
			name: "boolean_true_or_nothing",
			tokens: []lexer.Token{
				tok{[]byte("true"), lexer.TagTrue}, or, nothing, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					BooleanExpression{
						Left:      BooleanLiteral{Value: byteutil.True, Token: tok{[]byte("true"), lexer.TagTrue}},
						Right:     NothingLiteral{Token: nothing},
						Operation: OperationLiteral{Value: "or", Token: or},
					},
				},
			},
		},
		{
			name: "if_condition_nothing",
			tokens: []lexer.Token{
				tok{[]byte("if"), lexer.TagIf}, nothing,
				tok{[]byte("{"), lexer.TagOCurBrk}, one, semicolon, tok{[]byte("}"), lexer.TagCCurBrk},
				tok{[]byte("else"), lexer.TagElse},
				tok{[]byte("{"), lexer.TagOCurBrk}, two, semicolon, tok{[]byte("}"), lexer.TagCCurBrk},
				semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					IfExpression{
						Test: NothingLiteral{Token: nothing},
						Body: []Node{NumberLiteral{Value: 1, Token: one}},
						Else: &ElseExpression{
							Body: []Node{NumberLiteral{Value: 2, Token: two}},
						},
					},
				},
			},
		},
		{
			name: "if_body_nothing",
			tokens: []lexer.Token{
				tok{[]byte("if"), lexer.TagIf}, tok{[]byte("true"), lexer.TagTrue},
				tok{[]byte("{"), lexer.TagOCurBrk}, nothing, semicolon, tok{[]byte("}"), lexer.TagCCurBrk},
				tok{[]byte("else"), lexer.TagElse},
				tok{[]byte("{"), lexer.TagOCurBrk}, two, semicolon, tok{[]byte("}"), lexer.TagCCurBrk},
				semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					IfExpression{
						Test: BooleanLiteral{Value: byteutil.True, Token: tok{[]byte("true"), lexer.TagTrue}},
						Body: []Node{NothingLiteral{Token: nothing}},
						Else: &ElseExpression{
							Body: []Node{NumberLiteral{Value: 2, Token: two}},
						},
					},
				},
			},
		},
		{
			name: "branch_value_nothing",
			tokens: []lexer.Token{
				tok{[]byte("branch"), lexer.TagBranch}, tok{[]byte("{"), lexer.TagOCurBrk},
				tok{[]byte("true"), lexer.TagTrue}, tok{[]byte(":"), lexer.TagColon}, nothing, tok{[]byte(","), lexer.TagComma},
				three, semicolon,
				tok{[]byte("}"), lexer.TagCCurBrk}, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					IfExpression{
						Test: BooleanLiteral{Value: byteutil.True, Token: tok{[]byte("true"), lexer.TagTrue}},
						Body: []Node{NothingLiteral{Token: nothing}},
						Else: &ElseExpression{
							Body: []Node{NumberLiteral{Value: 3, Token: three}},
						},
					},
				},
			},
		},
		{
			name:   "print_nothing",
			tokens: []lexer.Token{tok{[]byte("print"), lexer.TagCallPrint}, nothing, semicolon, eof},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					PrintStatement{Param: NothingLiteral{Token: nothing}},
				},
			},
		},
		{
			name:   "echo_nothing",
			tokens: []lexer.Token{tok{[]byte("echo"), lexer.TagEcho}, nothing, semicolon, eof},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					EchoStatement{Param: NothingLiteral{Token: nothing}},
				},
			},
		},
		{
			name: "assert_nothing_condition_and_message",
			tokens: []lexer.Token{
				tok{[]byte("assert"), lexer.TagAssert}, tok{[]byte("("), lexer.TagOParen},
				nothing, tok{[]byte(","), lexer.TagComma}, nothing, tok{[]byte(")"), lexer.TagCParen}, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					AssertStatement{
						Condition: NothingLiteral{Token: nothing},
						Message:   NothingLiteral{Token: nothing},
						Token:     tok{[]byte("assert"), lexer.TagAssert},
					},
				},
			},
		},
		{
			name:   "unary_minus_nothing",
			tokens: []lexer.Token{tok{[]byte("-"), lexer.TagSub}, nothing, semicolon, eof},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					UnaryExpression{
						Expression: NothingLiteral{Token: nothing},
						Operation:  OperationLiteral{Value: "-", Token: sub},
					},
				},
			},
		},
		{
			name: "defer_body_nothing",
			tokens: []lexer.Token{
				tok{[]byte("defer"), lexer.TagDefer}, tok{[]byte("{"), lexer.TagOCurBrk},
				nothing, semicolon,
				tok{[]byte("}"), lexer.TagCCurBrk}, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					DeferExpression{
						Block: BlockExpression{
							Body: []Node{
								NothingLiteral{Token: nothing},
							},
						},
					},
				},
			},
		},
		{
			name: "call_argument_nothing",
			tokens: []lexer.Token{
				tok{[]byte("ident"), lexer.TagIdent}, tok{[]byte("f"), lexer.TagId},
				tok{[]byte("="), lexer.TagAssign},
				tok{[]byte("defer"), lexer.TagDefer}, tok{[]byte("{"), lexer.TagOCurBrk},
				tok{[]byte("arguments"), lexer.TagArguments}, tok{[]byte("("), lexer.TagOParen}, zero, tok{[]byte(")"), lexer.TagCParen}, semicolon,
				tok{[]byte("}"), lexer.TagCCurBrk}, semicolon,
				tok{[]byte("f"), lexer.TagId}, tok{[]byte("("), lexer.TagOParen}, nothing, tok{[]byte(")"), lexer.TagCParen}, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					IdentLiteral{
						Id:    "f",
						Token: tok{[]byte("f"), lexer.TagId},
						Value: DeferExpression{
							Block: BlockExpression{
								Body: []Node{
									ArgumentsExpression{
										Nth: NumberLiteral{Value: 0, Token: zero},
									},
								},
							},
						},
					},
					CalleeLiteral{
						Id:     IdentifierLiteral{Value: "f", Token: tok{[]byte("f"), lexer.TagId}},
						Params: []ParameterLiteral{{Expression: NothingLiteral{Token: nothing}}},
					},
				},
			},
		},
		{
			name: "tape_item_nothing",
			tokens: []lexer.Token{
				tok{[]byte("["), lexer.TagOBrk}, nothing, tok{[]byte(","), lexer.TagComma},
				one, tok{[]byte("]"), lexer.TagCBrk}, semicolon, eof,
			},
			want: &Module{
				Name: "main",
				Expressions: []Node{
					TapeBracketExpression{
						Items: []Node{
							NothingLiteral{Token: nothing},
							NumberLiteral{Value: 1, Token: one},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := NewParserOptions{}
			if c.name == "assert_nothing_condition_and_message" {
				opts.Filename = "a.test.ar"
			}
			p := New(c.tokens, opts)
			ast, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ModuleEqual(ast.Module, *c.want) {
				t.Errorf("AST mismatch:\ngot  %+v\nwant %+v", ast.Module, *c.want)
			}
		})
	}
}

func TestParseNamespacedCallAndUse(t *testing.T) {
	l := lexer.New(lexer.NewLexerOptions{})
	tokens, err := l.GetFilledTokens([]byte("std::fs::io::open_file(1);"))
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	p := New(tokens, NewParserOptions{})
	ast, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ast.Module.Expressions) != 1 {
		t.Fatalf("expected 1 expression, got %d", len(ast.Module.Expressions))
	}
	call, ok := ast.Module.Expressions[0].(CalleeLiteral)
	if !ok {
		t.Fatalf("expected CalleeLiteral, got %T", ast.Module.Expressions[0])
	}
	if len(call.Id.Namespace) != 3 || call.Id.Namespace[0] != "std" || call.Id.Namespace[1] != "fs" || call.Id.Namespace[2] != "io" {
		t.Errorf("expected Id.Namespace [std fs io], got %v", call.Id.Namespace)
	}
	if call.Id.Value != "open_file" {
		t.Errorf("expected Id.Value open_file, got %q", call.Id.Value)
	}
	if len(call.Params) != 1 {
		t.Errorf("expected 1 param, got %d", len(call.Params))
	}

	tokens2, err := l.GetFilledTokens([]byte("use std::fs::io as io;"))
	if err != nil {
		t.Fatalf("lex use: %v", err)
	}
	p2 := New(tokens2, NewParserOptions{})
	ast2, err := p2.Parse()
	if err != nil {
		t.Fatalf("parse use: %v", err)
	}
	if len(ast2.Module.Expressions) != 1 {
		t.Fatalf("expected 1 expression (use), got %d", len(ast2.Module.Expressions))
	}
	use, ok := ast2.Module.Expressions[0].(UseDeclaration)
	if !ok {
		t.Fatalf("expected UseDeclaration, got %T", ast2.Module.Expressions[0])
	}
	if len(use.Path) != 3 || use.Path[0] != "std" || use.Path[1] != "fs" || use.Path[2] != "io" {
		t.Errorf("expected Path [std fs io], got %v", use.Path)
	}
	if use.Alias != "io" {
		t.Errorf("expected Alias io, got %q", use.Alias)
	}
}
