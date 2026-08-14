package parser

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/lexer"
)

// These tests are about the shape of the tree, which nothing else checks. The evaluator
// suite runs source and compares bytes, so a node built with the wrong field passes there
// as long as the answer comes out right.

// parse compiles source and returns the top-level nodes.
func parse(t *testing.T, source string) []Node {
	t.Helper()
	ast, err := parseSource(t, source, "main.ar")
	if err != nil {
		t.Fatalf("parsing %q: %v", source, err)
	}
	return ast.Nodes
}

func parseSource(t *testing.T, source, filename string) (AST, error) {
	t.Helper()
	tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens([]byte(source))
	if err != nil {
		return AST{}, err
	}
	return New(NewParserOptions{Filename: filename, Tokens: tokens}).Parse()
}

// first returns the single top-level node, asserting the type.
func first[T Node](t *testing.T, source string) T {
	t.Helper()
	nodes := parse(t, source)
	if len(nodes) != 1 {
		t.Fatalf("%q produced %d nodes, want 1", source, len(nodes))
	}
	node, ok := nodes[0].(T)
	if !ok {
		var want T
		t.Fatalf("%q produced %T, want %T", source, nodes[0], want)
	}
	return node
}

func TestParseIdentShape(t *testing.T) {
	ident := first[IdentLiteral](t, "ident total = 42;")

	if ident.Id != "total" {
		t.Errorf("id = %q, want total", ident.Id)
	}
	number, ok := ident.Value.(NumberLiteral)
	if !ok {
		t.Fatalf("value is %T, want NumberLiteral", ident.Value)
	}
	if number.Value != 42 {
		t.Errorf("value = %d, want 42", number.Value)
	}
}

func TestParseNumberShape(t *testing.T) {
	cases := []struct {
		source string
		want   uint64
	}{
		{source: "7;", want: 7},
		{source: "1_000;", want: 1000},
		{source: "0xFF;", want: 255},
		{source: "0XFF;", want: 255},
		{source: "0x1A2B;", want: 6699},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			if got := first[NumberLiteral](t, tc.source); got.Value != tc.want {
				t.Errorf("value = %d, want %d", got.Value, tc.want)
			}
		})
	}
}

func TestParseBooleanShape(t *testing.T) {
	truth := first[BooleanLiteral](t, "true;")
	if want := byteutil.TrueTape(byteutil.DefaultTapeSize); string(truth.Value) != string(want) {
		t.Errorf("true = %v, want %v", truth.Value, want)
	}

	lie := first[BooleanLiteral](t, "false;")
	if want := byteutil.FalseTape(byteutil.DefaultTapeSize); string(lie.Value) != string(want) {
		t.Errorf("false = %v, want %v", lie.Value, want)
	}
}

// A reel is a run of tapes, one per character.
func TestParseReelShape(t *testing.T) {
	reel := first[ReelLiteral](t, `"hi";`)

	if len(reel.Value) != 2 {
		t.Fatalf("%q produced %d tapes, want 2", "hi", len(reel.Value))
	}
	if got := reel.Value[0][len(reel.Value[0])-1]; got != 'h' {
		t.Errorf("first tape ends in %d, want %d", got, 'h')
	}
	if got := reel.Value[1][len(reel.Value[1])-1]; got != 'i' {
		t.Errorf("second tape ends in %d, want %d", got, 'i')
	}

	empty := first[ReelLiteral](t, `"";`)
	if len(empty.Value) != 1 {
		t.Errorf("an empty reel has %d tapes, want 1", len(empty.Value))
	}
}

func TestParseTapeLiteralShape(t *testing.T) {
	tape := first[TapeBracketExpression](t, "[1, 2, 3];")
	if len(tape.Items) != 3 {
		t.Fatalf("got %d items, want 3", len(tape.Items))
	}

	empty := first[TapeBracketExpression](t, "[];")
	if len(empty.Items) != 0 {
		t.Errorf("got %d items, want none", len(empty.Items))
	}
}

func TestParseTapeOperationShapes(t *testing.T) {
	pull := first[PullExpression](t, "pull [1] 2;")
	if _, ok := pull.Target.(TapeBracketExpression); !ok {
		t.Errorf("pull target is %T, want a tape", pull.Target)
	}
	if _, ok := pull.Item.(NumberLiteral); !ok {
		t.Errorf("pull item is %T, want a number", pull.Item)
	}

	push := first[PushExpression](t, "push [1] 2;")
	if _, ok := push.Target.(TapeBracketExpression); !ok {
		t.Errorf("push target is %T, want a tape", push.Target)
	}

	head := first[HeadExpression](t, "head [1, 2, 3] 2;")
	if head.Length != 2 {
		t.Errorf("head length = %d, want 2", head.Length)
	}

	tail := first[TailExpression](t, "tail [1, 2, 3] 2;")
	if tail.Length != 2 {
		t.Errorf("tail length = %d, want 2", tail.Length)
	}
}

func TestParseDeferShape(t *testing.T) {
	deferred := first[DeferExpression](t, "defer { 1; 2; };")
	if len(deferred.Block.Body) != 2 {
		t.Errorf("body holds %d expressions, want 2", len(deferred.Block.Body))
	}

	empty := first[DeferExpression](t, "defer {};")
	if len(empty.Block.Body) != 0 {
		t.Errorf("an empty defer holds %d expressions, want none", len(empty.Block.Body))
	}
}

func TestParseBlockShape(t *testing.T) {
	block := first[BlockExpression](t, "{ 1; 2; 3; };")
	if len(block.Body) != 3 {
		t.Errorf("body holds %d expressions, want 3", len(block.Body))
	}

	empty := first[BlockExpression](t, "{};")
	if len(empty.Body) != 0 {
		t.Errorf("an empty block holds %d expressions, want none", len(empty.Body))
	}
}

func TestParseCalleeShape(t *testing.T) {
	call := first[CalleeLiteral](t, "sum(1, 2);")

	if call.Id.Value != "sum" {
		t.Errorf("callee = %q, want sum", call.Id.Value)
	}
	if len(call.Params) != 2 {
		t.Fatalf("got %d parameters, want 2", len(call.Params))
	}

	none := first[CalleeLiteral](t, "go();")
	if len(none.Params) != 0 {
		t.Errorf("got %d parameters, want none", len(none.Params))
	}

	// Without parentheses it is a plain identifier, not a call.
	if _, ok := parse(t, "sum;")[0].(IdentifierLiteral); !ok {
		t.Error("a bare name should parse as an identifier")
	}
}

func TestParseFeedShape(t *testing.T) {
	withParens := first[FeedExpression](t, "feed(2);")
	if withParens.Nth.Value != 2 {
		t.Errorf("index = %d, want 2", withParens.Nth.Value)
	}

	// The form without parentheses is also accepted.
	bare := first[FeedExpression](t, "feed 3;")
	if bare.Nth.Value != 3 {
		t.Errorf("index = %d, want 3", bare.Nth.Value)
	}
}

func TestParseIfShape(t *testing.T) {
	withoutElse := first[IfExpression](t, "if true { 1; };")
	if withoutElse.Else != nil {
		t.Error("there is no else here")
	}
	if len(withoutElse.Body) != 1 {
		t.Errorf("body holds %d expressions, want 1", len(withoutElse.Body))
	}
	if _, ok := withoutElse.Test.(BooleanLiteral); !ok {
		t.Errorf("test is %T, want a boolean", withoutElse.Test)
	}

	withElse := first[IfExpression](t, "if 1 bigger 2 { 1; } else { 2; };")
	if withElse.Else == nil {
		t.Fatal("the else is missing")
	}
	if len(withElse.Else.Body) != 1 {
		t.Errorf("else holds %d expressions, want 1", len(withElse.Else.Body))
	}
	if _, ok := withElse.Test.(RelativeExpression); !ok {
		t.Errorf("test is %T, want a comparison", withElse.Test)
	}
}

// branch is sugar: it desugars into nested ifs, with the fallback as the innermost else.
func TestParseBranchDesugarsIntoNestedIfs(t *testing.T) {
	outer := first[IfExpression](t, `branch {
  1 equals 1: 10,
  2 equals 2: 20,
  30;
};`)

	if outer.Else == nil {
		t.Fatal("the first branch item needs an else holding the rest")
	}
	inner, ok := outer.Else.Body[0].(IfExpression)
	if !ok {
		t.Fatalf("the else holds %T, want the next branch item", outer.Else.Body[0])
	}
	if inner.Else == nil {
		t.Fatal("the second item needs an else holding the fallback")
	}
	if _, ok := inner.Else.Body[0].(NumberLiteral); !ok {
		t.Errorf("the fallback is %T, want the number", inner.Else.Body[0])
	}
}

// The three print builtins differ only in how the value is read, so they parse into one
// node carrying which reading was asked for.
func TestParsePrintShapes(t *testing.T) {
	cases := []struct {
		source string
		want   PrintFormat
	}{
		{source: "printb 1;", want: PrintBytes},
		{source: `printc "hi";`, want: PrintChars},
		{source: "printd 1;", want: PrintDecimal},
	}

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			printed := first[PrintStatement](t, tc.source)
			if printed.Format != tc.want {
				t.Errorf("format = %q, want %q", printed.Format, tc.want)
			}
			if printed.Param == nil {
				t.Error("the value to print is missing")
			}
		})
	}
}

func TestParseAssertShape(t *testing.T) {
	ast, err := parseSource(t, `assert(1 equals 1, "ok");`, "checks.test.ar")
	if err != nil {
		t.Fatalf("assert in a test file: %v", err)
	}
	assertion, ok := ast.Nodes[0].(AssertStatement)
	if !ok {
		t.Fatalf("got %T, want AssertStatement", ast.Nodes[0])
	}
	if _, ok := assertion.Condition.(RelativeExpression); !ok {
		t.Errorf("condition is %T, want a comparison", assertion.Condition)
	}
	if _, ok := assertion.Message.(ReelLiteral); !ok {
		t.Errorf("message is %T, want a reel", assertion.Message)
	}
}

func TestParseUnaryShape(t *testing.T) {
	unary := first[UnaryExpression](t, "-5;")
	if unary.Operation.Value != "-" {
		t.Errorf("operation = %q, want -", unary.Operation.Value)
	}
	if _, ok := unary.Expression.(NumberLiteral); !ok {
		t.Errorf("operand is %T, want a number", unary.Expression)
	}
}

func TestParseOperatorShapes(t *testing.T) {
	arithmetic := []string{"+", "-", "*", "/", "^"}
	for _, op := range arithmetic {
		t.Run(op, func(t *testing.T) {
			expr := first[BinaryExpression](t, "1 "+op+" 2;")
			if expr.Operation.Value != op {
				t.Errorf("operation = %q, want %q", expr.Operation.Value, op)
			}
		})
	}

	comparisons := []string{"equals", "different", "bigger", "smaller"}
	for _, op := range comparisons {
		t.Run(op, func(t *testing.T) {
			expr := first[RelativeExpression](t, "1 "+op+" 2;")
			if expr.Operation.Value != op {
				t.Errorf("operation = %q, want %q", expr.Operation.Value, op)
			}
		})
	}

	for _, op := range []string{"and", "or"} {
		t.Run(op, func(t *testing.T) {
			expr := first[BooleanExpression](t, "true "+op+" false;")
			if expr.Operation.Value != op {
				t.Errorf("operation = %q, want %q", expr.Operation.Value, op)
			}
		})
	}
}

// Precedence and associativity live in the shape of the tree, so this is the only place
// they can be checked directly.
func TestPrecedence(t *testing.T) {
	// 2 + 3 * 4 groups as 2 + (3 * 4)
	sum := first[BinaryExpression](t, "2 + 3 * 4;")
	if sum.Operation.Value != "+" {
		t.Fatalf("the outer operation is %q, want +", sum.Operation.Value)
	}
	product, ok := sum.Right.(BinaryExpression)
	if !ok || product.Operation.Value != "*" {
		t.Errorf("the right side is %T, want the multiplication", sum.Right)
	}

	// Parentheses override it: (2 + 3) * 4
	product = first[BinaryExpression](t, "(2 + 3) * 4;")
	if product.Operation.Value != "*" {
		t.Fatalf("the outer operation is %q, want *", product.Operation.Value)
	}
	if inner, ok := product.Left.(BinaryExpression); !ok || inner.Operation.Value != "+" {
		t.Errorf("the left side is %T, want the sum", product.Left)
	}

	// Comparison binds looser than arithmetic: (1 + 1) equals 2
	comparison := first[RelativeExpression](t, "1 + 1 equals 2;")
	if _, ok := comparison.Left.(BinaryExpression); !ok {
		t.Errorf("the left side is %T, want the sum", comparison.Left)
	}
}

// Additive expressions are left-associative, which is why the EVM lowering has to reorder
// them: 10 - 3 - 2 is (10 - 3) - 2, not 10 - (3 - 2).
func TestAdditiveIsLeftAssociative(t *testing.T) {
	expr := first[BinaryExpression](t, "10 - 3 - 2;")

	left, ok := expr.Left.(BinaryExpression)
	if !ok {
		t.Fatalf("the left side is %T, want the inner subtraction", expr.Left)
	}
	if first, ok := left.Left.(NumberLiteral); !ok || first.Value != 10 {
		t.Errorf("the innermost operand is %v, want 10", left.Left)
	}
	if last, ok := expr.Right.(NumberLiteral); !ok || last.Value != 2 {
		t.Errorf("the outer right operand is %v, want 2", expr.Right)
	}
}

// Multiplicative expressions are left-associative too: 20 / 5 / 2 is (20 / 5) / 2, not
// 20 / (5 / 2). They used to group to the right, which answered 10 for that.
func TestMultiplicativeIsLeftAssociative(t *testing.T) {
	for _, source := range []string{"20 / 5 / 2;", "4 / 2 * 6;", "2 * 3 / 6;"} {
		expr := first[BinaryExpression](t, source)

		if _, ok := expr.Left.(BinaryExpression); !ok {
			t.Errorf("%s: the left side is %T, want the inner operation", source, expr.Left)
		}
		if _, ok := expr.Right.(NumberLiteral); !ok {
			t.Errorf("%s: the right side is %T, want the last operand alone", source, expr.Right)
		}
	}
}

// Exponentiation recurses to the right: 2 ^ 3 ^ 2 is 2 ^ (3 ^ 2).
func TestExponentiationIsRightAssociative(t *testing.T) {
	expr := first[BinaryExpression](t, "2 ^ 3 ^ 2;")
	if _, ok := expr.Right.(BinaryExpression); !ok {
		t.Errorf("the right side is %T, want the inner exponentiation", expr.Right)
	}
}

func TestParseSeveralTopLevelExpressions(t *testing.T) {
	nodes := parse(t, "ident a = 1;\nprintb a;\na + 1;\n")
	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}
	if _, ok := nodes[0].(IdentLiteral); !ok {
		t.Errorf("first node is %T", nodes[0])
	}
	if _, ok := nodes[1].(PrintStatement); !ok {
		t.Errorf("second node is %T", nodes[1])
	}
	if _, ok := nodes[2].(BinaryExpression); !ok {
		t.Errorf("third node is %T", nodes[2])
	}
}

func TestParseEmptySource(t *testing.T) {
	if nodes := parse(t, ""); len(nodes) != 0 {
		t.Errorf("got %d nodes, want none", len(nodes))
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		filename string
		wantErr  string
	}{
		{name: "missing semicolon", source: "ident a = 1", wantErr: "unexpected token"},
		{name: "ident without a name", source: "ident = 1;", wantErr: "unexpected token"},
		{name: "unclosed block", source: "{ 1;", wantErr: "unexpected token"},
		{name: "unclosed parentheses", source: "(1 + 2;", wantErr: "unexpected token"},
		{name: "assert outside a test file", source: `assert(1 equals 1, "x");`, filename: "main.ar", wantErr: ".test.ar"},
		{name: "tape value over a byte", source: "[300];", wantErr: "between 0 and 255"},
		{name: "branch without a condition", source: "branch { 1: 2, 3; };", wantErr: "boolean expression"},
		{name: "pull with an invalid target", source: "pull true 1;", wantErr: "not a valid append target"},
		{name: "push with an invalid item", source: "push [1] true;", wantErr: "not a valid push item"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filename := tc.filename
			if filename == "" {
				filename = "main.ar"
			}
			_, err := parseSource(t, tc.source, filename)
			if err == nil {
				t.Fatalf("%q should not parse", tc.source)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

// Errors carry their position so the language server can underline them.
func TestParseErrorsArePositioned(t *testing.T) {
	_, err := parseSource(t, "ident a = 1;\nident = 2;\n", "main.ar")
	if err == nil {
		t.Fatal("expected an error")
	}

	perr, ok := err.(*lexer.Error)
	if !ok {
		t.Fatalf("error is %T, want *lexer.Error", err)
	}
	if perr.Line != 2 {
		t.Errorf("line = %d, want 2", perr.Line)
	}
	if perr.Offset == 0 {
		t.Error("offset is zero, so the server cannot place it")
	}
}

// The logger only runs with logging on, and it walks the tree it was given — a shape it
// cannot handle would panic in front of whoever turned the flag on.
func TestParseWithLoggingEnabled(t *testing.T) {
	source := `ident a = 1;
ident f = defer { feed(0) + 1; };
if a bigger 0 { printb f(a); } else { printc "no"; };
ident t = pull [1, 2] 3;
`
	tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}

	ast, err := New(NewParserOptions{
		Filename:      "main.ar",
		Tokens:        tokens,
		EnableLogging: true,
	}).Parse()
	if err != nil {
		t.Fatalf("parsing with logging on: %v", err)
	}
	if len(ast.Nodes) != 4 {
		t.Errorf("got %d nodes, want 4", len(ast.Nodes))
	}
}

// ASTEqual is what other packages use to compare trees, so it has to notice a difference
// wherever it sits.
func TestASTEqual(t *testing.T) {
	build := func(source string) AST {
		t.Helper()
		ast, err := parseSource(t, source, "main.ar")
		if err != nil {
			t.Fatalf("parsing %q: %v", source, err)
		}
		return ast
	}

	same := "ident a = defer { 1; };"
	if !ASTEqual(build(same), build(same)) {
		t.Error("the same source produced trees that do not compare equal")
	}

	cases := []struct {
		name string
		a, b string
	}{
		{name: "different values", a: "ident a = 1;", b: "ident a = 2;"},
		{name: "different names", a: "ident a = 1;", b: "ident b = 1;"},
		{name: "different operators", a: "1 + 2;", b: "1 - 2;"},
		{name: "different defer bodies", a: "defer { 1; };", b: "defer { 2; };"},
		{name: "different call arguments", a: "f(1);", b: "f(2);"},
		{name: "different node counts", a: "1;", b: "1;\n2;"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if ASTEqual(build(tc.a), build(tc.b)) {
				t.Errorf("%q and %q should not compare equal", tc.a, tc.b)
			}
		})
	}
}
