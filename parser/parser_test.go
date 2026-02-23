package parser

import (
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
	units := []ParserUnit{
		{
			Filename:  "test.ar",
			Namespace: "testing",
			Tokens:    tokens,
		},
	}
	p := &pr{cursor: 0, units: units}
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
	units := []ParserUnit{
		{
			Filename:  "test.ar",
			Namespace: "testing",
			Tokens:    tokens,
		},
	}
	p := New(NewParserOptions{
		Units: units,
	})
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
		units := []ParserUnit{
			{
				Filename:  "test.ar",
				Namespace: "testing",
				Tokens:    c.Tokens,
			},
		}
		p := New(NewParserOptions{
			Units: units,
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

func TestSetUnitIndexAndGetCurrentUnit(t *testing.T) {
	units := []ParserUnit{
		{Filename: "a.ar", Namespace: "ns_a", Tokens: []lexer.Token{}},
		{Filename: "b.ar", Namespace: "ns_b", Tokens: []lexer.Token{}},
		{Filename: "c.ar", Namespace: "ns_c", Tokens: []lexer.Token{}},
	}
	p := &pr{units: units, unitindex: 0}

	// GetCurrentUnit with index 0 returns first unit
	if got := p.GetCurrentUnit(); got == nil || got.Filename != "a.ar" || got.Namespace != "ns_a" {
		t.Errorf("GetCurrentUnit() at index 0: got %+v, want Filename=a.ar Namespace=ns_a", got)
	}

	// SetUnitIndex(1) then GetCurrentUnit returns second unit
	p.SetUnitIndex(1)
	if got := p.GetCurrentUnit(); got == nil || got.Filename != "b.ar" || got.Namespace != "ns_b" {
		t.Errorf("GetCurrentUnit() after SetUnitIndex(1): got %+v, want Filename=b.ar Namespace=ns_b", got)
	}

	// SetUnitIndex(2) then GetCurrentUnit returns third unit
	p.SetUnitIndex(2)
	if got := p.GetCurrentUnit(); got == nil || got.Filename != "c.ar" || got.Namespace != "ns_c" {
		t.Errorf("GetCurrentUnit() after SetUnitIndex(2): got %+v, want Filename=c.ar Namespace=ns_c", got)
	}

	// SetUnitIndex past end: GetCurrentUnit returns nil
	p.SetUnitIndex(3)
	if got := p.GetCurrentUnit(); got != nil {
		t.Errorf("GetCurrentUnit() with index >= len(units): got %+v, want nil", got)
	}
	p.SetUnitIndex(10)
	if got := p.GetCurrentUnit(); got != nil {
		t.Errorf("GetCurrentUnit() with index 10: got %+v, want nil", got)
	}

	// No units: GetCurrentUnit returns nil (unitindex 0 >= len(units)=0)
	pEmpty := &pr{units: []ParserUnit{}, unitindex: 0}
	if got := pEmpty.GetCurrentUnit(); got != nil {
		t.Errorf("GetCurrentUnit() with no units: got %+v, want nil", got)
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
	expected := Namespace{
		Name:         "testing",
		Dependencies: []string{},
		AST: []Node{
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
	}
	units := []ParserUnit{
		{
			Filename:  "test.ar",
			Namespace: "testing",
			Tokens:    tokens,
		},
	}
	p := &pr{cursor: 0, units: units}
	ast, err := p.Parse()
	if err != nil {
		t.Errorf("param: %v, %v", tokens, err)
	}
	if !NamespaceEqual(ast, expected) {
		t.Errorf("\nexpected: %+v,\ngot: %+v", expected, ast)
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
		want   *Namespace
	}{
		{
			name:   "top_level",
			tokens: []lexer.Token{nothing, semicolon, eof},
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
					NothingLiteral{Token: nothing},
				},
			},
		},
		{
			name: "binary_left_nothing_plus_one",
			tokens: []lexer.Token{
				nothing, sum, one, semicolon, eof,
			},
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
					PrintStatement{Param: NothingLiteral{Token: nothing}},
				},
			},
		},
		{
			name:   "echo_nothing",
			tokens: []lexer.Token{tok{[]byte("echo"), lexer.TagEcho}, nothing, semicolon, eof},
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			want: &Namespace{
				Name: "testing",
				AST: []Node{
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
			opts := NewParserOptions{
				Units: []ParserUnit{
					{
						Filename:  "test.ar",
						Namespace: "testing",
						Tokens:    c.tokens,
					},
				},
			}
			if c.name == "assert_nothing_condition_and_message" {
				opts.Units[0].Filename = "a.test.ar"
			}
			p := New(opts)
			ast, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !NamespaceEqual(ast, *c.want) {
				t.Errorf("AST mismatch:\ngot  %+v\nwant %+v", ast, *c.want)
			}
		})
	}
}

func TestParseUseDeclaration(t *testing.T) {
	semicolon := tok{[]byte(";"), lexer.TagSemicolon}
	eof := tok{[]byte(""), lexer.TagEOF}
	use := tok{[]byte("use"), lexer.TagUse}
	as := tok{[]byte("as"), lexer.TagAs}

	cases := []struct {
		name   string
		tokens []lexer.Token
		want   *Namespace
	}{
		{
			name: "use_single_segment_as_alias",
			tokens: []lexer.Token{
				use, tok{[]byte("math"), lexer.TagId},
				as, tok{[]byte("m"), lexer.TagId},
				semicolon, eof,
			},
			want: &Namespace{
				Name: "testing",
				AST: []Node{
					UseDeclaration{Namespace: "math", Alias: "m", Token: use},
				},
			},
		},
		{
			name: "use_namespaced_path_as_alias",
			tokens: []lexer.Token{
				use,
				tok{[]byte("std"), lexer.TagId},
				tok{[]byte("::"), lexer.TagNsScope},
				tok{[]byte("fs"), lexer.TagId},
				tok{[]byte("::"), lexer.TagNsScope},
				tok{[]byte("io"), lexer.TagId},
				as,
				tok{[]byte("io"), lexer.TagId},
				semicolon, eof,
			},
			want: &Namespace{
				Name: "testing",
				AST: []Node{
					UseDeclaration{Namespace: "std::fs::io", Alias: "io", Token: use},
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := New(NewParserOptions{
				Units: []ParserUnit{
					{
						Filename:  "test.ar",
						Namespace: "testing",
						Tokens:    c.tokens,
					},
				},
			})
			got, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !NamespaceEqual(got, *c.want) {
				t.Errorf("AST mismatch:\ngot  %+v\nwant %+v", got, *c.want)
			}
		})
	}
}

func TestParseNamespacedIdentifier(t *testing.T) {
	comma := tok{[]byte(","), lexer.TagComma}
	semicolon := tok{[]byte(";"), lexer.TagSemicolon}
	eof := tok{[]byte(""), lexer.TagEOF}
	one := tok{[]byte("1"), lexer.TagNumber}
	two := tok{[]byte("2"), lexer.TagNumber}

	cases := []struct {
		name   string
		tokens []lexer.Token
		want   *Namespace
	}{
		{
			name: "single_identifier",
			tokens: []lexer.Token{
				tok{[]byte("a"), lexer.TagId},
				semicolon, eof,
			},
			want: &Namespace{
				Name: "testing",
				AST: []Node{
					IdentifierLiteral{Value: "a", Token: tok{[]byte("a"), lexer.TagId}},
				},
			},
		},
		{
			name: "namespaced_identifier",
			tokens: []lexer.Token{
				tok{[]byte("a"), lexer.TagId}, tok{[]byte("::"), lexer.TagNsScope}, tok{[]byte("b"), lexer.TagId},
				semicolon, eof,
			},
			want: &Namespace{
				Name: "testing",
				AST: []Node{
					IdentifierLiteral{Value: "b", Namespace: "a", Token: tok{[]byte("b"), lexer.TagId}},
				},
			},
		},
		{
			name: "namespaced_identifier_with_defer",
			tokens: []lexer.Token{
				tok{[]byte("a"), lexer.TagId}, tok{[]byte("::"), lexer.TagNsScope},
				tok{[]byte("b"), lexer.TagId}, tok{[]byte("::"), lexer.TagNsScope},
				tok{[]byte("c"), lexer.TagId}, tok{[]byte("("), lexer.TagOParen},
				one, comma, two,
				tok{[]byte(")"), lexer.TagCParen},
				semicolon, eof,
			},
			want: &Namespace{
				Name: "testing",
				AST: []Node{
					CalleeLiteral{
						Id: IdentifierLiteral{Value: "c", Namespace: "a::b", Token: tok{[]byte("c"), lexer.TagId}},
						Params: []ParameterLiteral{
							{Expression: NumberLiteral{Value: 1, Token: one}},
							{Expression: NumberLiteral{Value: 2, Token: two}},
						},
					},
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := NewParserOptions{
				Units: []ParserUnit{
					{
						Filename:  "test.ar",
						Namespace: "testing",
						Tokens:    c.tokens,
					},
				},
			}
			p := New(opts)
			got, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !NamespaceEqual(got, *c.want) {
				t.Errorf("AST mismatch:\ngot  %+v\nwant %+v", got, *c.want)
			}
		})
	}
}
