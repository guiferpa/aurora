package ast

import (
	"testing"

	"github.com/guiferpa/aurora/wire/token"
)

// nodeEqual is what every parser test believes: ASTEqual asks it whether the tree that came
// out of the parser is the tree that was expected. Nothing ever asked nodeEqual anything, so
// a comparison answering "equal" too easily would have made all of those tests agree with
// almost any tree — and said nothing while doing it.
//
// Every case below is a pair that must not be confused: the same kind of node, one field
// apart. A comparison that forgets a field passes the alike case and fails here.

// number, ident and word build the small nodes the bigger cases are made of.
func number(v uint64) NumberLiteral       { return NumberLiteral{Value: v} }
func word(v string) IdentifierLiteral     { return IdentifierLiteral{Value: v} }
func operation(v string) OperationLiteral { return OperationLiteral{Value: v} }

var (
	one = token.New([]byte("1"), token.TagNumber, 0, 0, 0)
	two = token.New([]byte("2"), token.TagNumber, 0, 0, 0)
	// Same text, different tag: TokenEqual reads both, and a token is not only its match.
	oneAsIdent = token.New([]byte("1"), token.TagIdent, 0, 0, 0)
)

var comparisonCases = []struct {
	name string
	a, b Node
	want bool
}{
	{name: "nothing against nothing", a: nil, b: nil, want: true},
	{name: "nothing against a node", a: nil, b: number(1), want: false},
	{name: "a node against nothing", a: number(1), b: nil, want: false},
	{name: "two kinds of node", a: number(1), b: BooleanLiteral{Value: []byte{1}}, want: false},

	{name: "number, alike", a: number(1), b: number(1), want: true},
	{name: "number, another value", a: number(1), b: number(2), want: false},
	{
		name: "number, another token",
		a:    NumberLiteral{Value: 1, Token: one},
		b:    NumberLiteral{Value: 1, Token: two},
		want: false,
	},

	{
		name: "boolean, alike",
		a:    BooleanLiteral{Value: []byte{1}, Token: one},
		b:    BooleanLiteral{Value: []byte{1}, Token: one},
		want: true,
	},
	{
		name: "boolean, another value",
		a:    BooleanLiteral{Value: []byte{1}},
		b:    BooleanLiteral{Value: []byte{0}},
		want: false,
	},
	{
		name: "boolean, another token",
		a:    BooleanLiteral{Value: []byte{1}, Token: one},
		b:    BooleanLiteral{Value: []byte{1}, Token: oneAsIdent},
		want: false,
	},

	{
		name: "binary, alike",
		a:    BinaryExpression{Left: number(1), Right: number(2), Operation: operation("+")},
		b:    BinaryExpression{Left: number(1), Right: number(2), Operation: operation("+")},
		want: true,
	},
	{
		name: "binary, another left",
		a:    BinaryExpression{Left: number(1), Right: number(2), Operation: operation("+")},
		b:    BinaryExpression{Left: number(3), Right: number(2), Operation: operation("+")},
		want: false,
	},
	{
		name: "binary, another right",
		a:    BinaryExpression{Left: number(1), Right: number(2), Operation: operation("+")},
		b:    BinaryExpression{Left: number(1), Right: number(3), Operation: operation("+")},
		want: false,
	},
	{
		// The operands are the same and the tree is not: this is the pair that catches a
		// comparison ignoring the operator.
		name: "binary, another operation",
		a:    BinaryExpression{Left: number(1), Right: number(2), Operation: operation("+")},
		b:    BinaryExpression{Left: number(1), Right: number(2), Operation: operation("-")},
		want: false,
	},

	{
		name: "relative, another operation",
		a:    RelativeExpression{Left: number(1), Right: number(2), Operation: operation("bigger")},
		b:    RelativeExpression{Left: number(1), Right: number(2), Operation: operation("smaller")},
		want: false,
	},
	{
		name: "relative, alike",
		a:    RelativeExpression{Left: number(1), Right: number(2), Operation: operation("bigger")},
		b:    RelativeExpression{Left: number(1), Right: number(2), Operation: operation("bigger")},
		want: true,
	},

	{
		name: "boolean expression, another operation",
		a:    BooleanExpression{Left: number(1), Right: number(2), Operation: operation("and")},
		b:    BooleanExpression{Left: number(1), Right: number(2), Operation: operation("or")},
		want: false,
	},
	{
		name: "boolean expression, alike",
		a:    BooleanExpression{Left: number(1), Right: number(2), Operation: operation("and")},
		b:    BooleanExpression{Left: number(1), Right: number(2), Operation: operation("and")},
		want: true,
	},

	{
		name: "unary, alike",
		a:    UnaryExpression{Expression: number(1), Operation: operation("-")},
		b:    UnaryExpression{Expression: number(1), Operation: operation("-")},
		want: true,
	},
	{
		// The operator dropped here once made -5 into 5, which is why it is asked twice.
		name: "unary, another operation",
		a:    UnaryExpression{Expression: number(1), Operation: operation("-")},
		b:    UnaryExpression{Expression: number(1), Operation: operation("!")},
		want: false,
	},
	{
		name: "unary, another expression",
		a:    UnaryExpression{Expression: number(1), Operation: operation("-")},
		b:    UnaryExpression{Expression: number(2), Operation: operation("-")},
		want: false,
	},

	{
		name: "if, alike",
		a:    IfExpression{Test: number(1), Body: []Node{number(2)}},
		b:    IfExpression{Test: number(1), Body: []Node{number(2)}},
		want: true,
	},
	{
		name: "if, another test",
		a:    IfExpression{Test: number(1), Body: []Node{number(2)}},
		b:    IfExpression{Test: number(3), Body: []Node{number(2)}},
		want: false,
	},
	{
		name: "if, another body",
		a:    IfExpression{Test: number(1), Body: []Node{number(2)}},
		b:    IfExpression{Test: number(1), Body: []Node{number(3)}},
		want: false,
	},
	{
		name: "if, a shorter body",
		a:    IfExpression{Test: number(1), Body: []Node{number(2), number(3)}},
		b:    IfExpression{Test: number(1), Body: []Node{number(2)}},
		want: false,
	},
	{
		name: "if, one of them with an else",
		a:    IfExpression{Test: number(1), Else: &ElseExpression{Body: []Node{number(2)}}},
		b:    IfExpression{Test: number(1)},
		want: false,
	},
	{
		name: "if, another else",
		a:    IfExpression{Test: number(1), Else: &ElseExpression{Body: []Node{number(2)}}},
		b:    IfExpression{Test: number(1), Else: &ElseExpression{Body: []Node{number(3)}}},
		want: false,
	},
	{
		name: "if, the same else",
		a:    IfExpression{Test: number(1), Else: &ElseExpression{Body: []Node{number(2)}}},
		b:    IfExpression{Test: number(1), Else: &ElseExpression{Body: []Node{number(2)}}},
		want: true,
	},

	{
		name: "block, alike",
		a:    BlockExpression{Body: []Node{number(1)}},
		b:    BlockExpression{Body: []Node{number(1)}},
		want: true,
	},
	{
		name: "block, another body",
		a:    BlockExpression{Body: []Node{number(1)}},
		b:    BlockExpression{Body: []Node{number(2)}},
		want: false,
	},

	{
		name: "defer, alike",
		a:    DeferExpression{Block: BlockExpression{Body: []Node{number(1)}}},
		b:    DeferExpression{Block: BlockExpression{Body: []Node{number(1)}}},
		want: true,
	},
	{
		name: "defer, another block",
		a:    DeferExpression{Block: BlockExpression{Body: []Node{number(1)}}},
		b:    DeferExpression{Block: BlockExpression{Body: []Node{number(2)}}},
		want: false,
	},

	{
		name: "ident, alike",
		a:    IdentLiteral{Id: "x", Token: one, Value: number(1)},
		b:    IdentLiteral{Id: "x", Token: one, Value: number(1)},
		want: true,
	},
	{
		name: "ident, another name",
		a:    IdentLiteral{Id: "x", Value: number(1)},
		b:    IdentLiteral{Id: "y", Value: number(1)},
		want: false,
	},
	{
		name: "ident, another value",
		a:    IdentLiteral{Id: "x", Value: number(1)},
		b:    IdentLiteral{Id: "x", Value: number(2)},
		want: false,
	},
	{
		name: "ident, another token",
		a:    IdentLiteral{Id: "x", Token: one, Value: number(1)},
		b:    IdentLiteral{Id: "x", Token: two, Value: number(1)},
		want: false,
	},

	{
		name: "identifier, alike",
		a:    IdentifierLiteral{Value: "x", Token: one},
		b:    IdentifierLiteral{Value: "x", Token: one},
		want: true,
	},
	{name: "identifier, another name", a: word("x"), b: word("y"), want: false},
	{
		name: "identifier, another token",
		a:    IdentifierLiteral{Value: "x", Token: one},
		b:    IdentifierLiteral{Value: "x", Token: two},
		want: false,
	},

	{name: "operation, alike", a: operation("+"), b: operation("+"), want: true},
	{name: "operation, another value", a: operation("+"), b: operation("-"), want: false},

	{
		// The three prints differ only in how the value is read, so the format is the whole
		// difference between printb and printd.
		name: "print, another format",
		a:    PrintStatement{Format: PrintBytes, Param: number(1)},
		b:    PrintStatement{Format: PrintDecimal, Param: number(1)},
		want: false,
	},
	{
		name: "print, alike",
		a:    PrintStatement{Format: PrintChars, Param: number(1)},
		b:    PrintStatement{Format: PrintChars, Param: number(1)},
		want: true,
	},
	{
		name: "print, another parameter",
		a:    PrintStatement{Format: PrintChars, Param: number(1)},
		b:    PrintStatement{Format: PrintChars, Param: number(2)},
		want: false,
	},

	{
		name: "assert, alike",
		a:    AssertStatement{Condition: number(1), Message: "holds", Token: one},
		b:    AssertStatement{Condition: number(1), Message: "holds", Token: one},
		want: true,
	},
	{
		name: "assert, another message",
		a:    AssertStatement{Condition: number(1), Message: "holds"},
		b:    AssertStatement{Condition: number(1), Message: "does not"},
		want: false,
	},
	{
		name: "assert, another condition",
		a:    AssertStatement{Condition: number(1), Message: "holds"},
		b:    AssertStatement{Condition: number(2), Message: "holds"},
		want: false,
	},

	{
		name: "struct declaration, alike",
		a:    StructDeclaration{Name: "Point", Fields: []string{"x", "y"}},
		b:    StructDeclaration{Name: "Point", Fields: []string{"x", "y"}},
		want: true,
	},
	{
		name: "struct declaration, another name",
		a:    StructDeclaration{Name: "Point", Fields: []string{"x", "y"}},
		b:    StructDeclaration{Name: "Pair", Fields: []string{"x", "y"}},
		want: false,
	},
	{
		// The fields are positional, so their order is part of the shape.
		name: "struct declaration, the fields the other way round",
		a:    StructDeclaration{Name: "Point", Fields: []string{"x", "y"}},
		b:    StructDeclaration{Name: "Point", Fields: []string{"y", "x"}},
		want: false,
	},

	{
		name: "use declaration, alike",
		a:    UseDeclaration{Specifier: "a/b/c", Alias: "x"},
		b:    UseDeclaration{Specifier: "a/b/c", Alias: "x"},
		want: true,
	},
	{
		name: "use declaration, another module",
		a:    UseDeclaration{Specifier: "a/b/c", Alias: "x"},
		b:    UseDeclaration{Specifier: "a/b/d", Alias: "x"},
		want: false,
	},
	{
		// The alias belongs to the file that wrote it, so it is part of the declaration and
		// not a way of spelling the same one.
		name: "use declaration, another alias",
		a:    UseDeclaration{Specifier: "a/b/c", Alias: "x"},
		b:    UseDeclaration{Specifier: "a/b/c", Alias: "y"},
		want: false,
	},

	{
		name: "struct literal, alike",
		a:    StructLiteral{Name: "Point", Values: []Node{number(1), number(2)}},
		b:    StructLiteral{Name: "Point", Values: []Node{number(1), number(2)}},
		want: true,
	},
	{
		name: "struct literal, another name",
		a:    StructLiteral{Name: "Point", Values: []Node{number(1)}},
		b:    StructLiteral{Name: "Pair", Values: []Node{number(1)}},
		want: false,
	},
	{
		name: "struct literal, another value",
		a:    StructLiteral{Name: "Point", Values: []Node{number(1), number(2)}},
		b:    StructLiteral{Name: "Point", Values: []Node{number(1), number(3)}},
		want: false,
	},
	{
		name: "struct literal, one value fewer",
		a:    StructLiteral{Name: "Point", Values: []Node{number(1), number(2)}},
		b:    StructLiteral{Name: "Point", Values: []Node{number(1)}},
		want: false,
	},

	{
		name: "field, alike",
		a:    FieldExpression{Expression: word("p"), Index: 1},
		b:    FieldExpression{Expression: word("p"), Index: 1},
		want: true,
	},
	{
		name: "field, another index",
		a:    FieldExpression{Expression: word("p"), Index: 0},
		b:    FieldExpression{Expression: word("p"), Index: 1},
		want: false,
	},
	{
		name: "field, another expression",
		a:    FieldExpression{Expression: word("p"), Index: 1},
		b:    FieldExpression{Expression: word("q"), Index: 1},
		want: false,
	},

	{
		name: "shaped, alike",
		a:    ShapedExpression{Expression: word("v"), Struct: "Point"},
		b:    ShapedExpression{Expression: word("v"), Struct: "Point"},
		want: true,
	},
	{
		name: "shaped, another struct",
		a:    ShapedExpression{Expression: word("v"), Struct: "Point"},
		b:    ShapedExpression{Expression: word("v"), Struct: "Pair"},
		want: false,
	},
	{
		name: "shaped, another expression",
		a:    ShapedExpression{Expression: word("v"), Struct: "Point"},
		b:    ShapedExpression{Expression: word("w"), Struct: "Point"},
		want: false,
	},

	{
		name: "callee, alike",
		a:    CalleeLiteral{Id: word("f"), Params: []ParameterLiteral{{Expression: number(1)}}},
		b:    CalleeLiteral{Id: word("f"), Params: []ParameterLiteral{{Expression: number(1)}}},
		want: true,
	},
	{
		name: "callee, another name",
		a:    CalleeLiteral{Id: word("f"), Params: []ParameterLiteral{{Expression: number(1)}}},
		b:    CalleeLiteral{Id: word("g"), Params: []ParameterLiteral{{Expression: number(1)}}},
		want: false,
	},
	{
		name: "callee, another parameter",
		a:    CalleeLiteral{Id: word("f"), Params: []ParameterLiteral{{Expression: number(1)}}},
		b:    CalleeLiteral{Id: word("f"), Params: []ParameterLiteral{{Expression: number(2)}}},
		want: false,
	},
	{
		name: "callee, one parameter fewer",
		a:    CalleeLiteral{Id: word("f"), Params: []ParameterLiteral{{Expression: number(1)}, {Expression: number(2)}}},
		b:    CalleeLiteral{Id: word("f"), Params: []ParameterLiteral{{Expression: number(1)}}},
		want: false,
	},

	{
		name: "tape, alike",
		a:    TapeBracketExpression{Items: []Node{number(1), number(2)}},
		b:    TapeBracketExpression{Items: []Node{number(1), number(2)}},
		want: true,
	},
	{
		name: "tape, another item",
		a:    TapeBracketExpression{Items: []Node{number(1), number(2)}},
		b:    TapeBracketExpression{Items: []Node{number(1), number(3)}},
		want: false,
	},
	{
		name: "tape, one item fewer",
		a:    TapeBracketExpression{Items: []Node{number(1), number(2)}},
		b:    TapeBracketExpression{Items: []Node{number(1)}},
		want: false,
	},

	{
		name: "feed, alike",
		a:    FeedExpression{Nth: NumberLiteral{Value: 0, Token: one}},
		b:    FeedExpression{Nth: NumberLiteral{Value: 0, Token: one}},
		want: true,
	},
	{
		name: "feed, another position",
		a:    FeedExpression{Nth: number(0)},
		b:    FeedExpression{Nth: number(1)},
		want: false,
	},

	// A node the switch does not name falls to reflect.DeepEqual, which is what keeps a new
	// node comparable before anyone writes a case for it.
	{
		name: "text, alike",
		a:    TextLiteral{Value: []byte("hi"), Token: one},
		b:    TextLiteral{Value: []byte("hi"), Token: one},
		want: true,
	},
	{
		name: "text, another value",
		a:    TextLiteral{Value: []byte("hi")},
		b:    TextLiteral{Value: []byte("ho")},
		want: false,
	},
}

func TestNodeEqualTellsNodesApart(t *testing.T) {
	for _, tc := range comparisonCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nodeEqual(tc.a, tc.b); got != tc.want {
				t.Errorf("answered %v, want %v", got, tc.want)
			}
			// Comparing is symmetric: reading the arguments the other way round is the same
			// question, and a case that only looks at the first one would answer differently.
			if got := nodeEqual(tc.b, tc.a); got != tc.want {
				t.Errorf("answered %v the other way round, want %v", got, tc.want)
			}
		})
	}
}

// ASTEqual compares the file as well as the trees, and the trees are what TestASTEqual in
// shapes_test.go already asks about through real sources. The name is the half nothing
// reached: two files holding the same expressions are still two files.
func TestASTEqualComparesTheFilename(t *testing.T) {
	a := AST{Filename: "main.ar", Nodes: []Node{number(1)}}
	b := AST{Filename: "other.ar", Nodes: []Node{number(1)}}

	if !Equal(a, a) {
		t.Error("a file does not compare equal to itself")
	}
	if Equal(a, b) {
		t.Error("two filenames compared equal")
	}
}
