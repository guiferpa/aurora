package cli

import "testing"

// The precedence ladder of the language, loosest binding at the top:
//
//	or
//	and
//	equals  different  bigger  smaller
//	+  -
//	*  /
//	^
//	-  (unary)
//	.  as  (postfix)
//	numbers, names, calls, parentheses
//
// Two bugs lived in this ladder, and both were invisible until a program gave a wrong
// answer: `*` and `/` grouped to the right, and `and` shared a rung with `or`. Neither
// shows up in an expression with one operator, which is what most tests are made of.
//
// So each rung is pinned three ways: against the rung below it, on its own for grouping,
// and with parentheses to prove the grouping is what parentheses override. Every expected
// value here was read off a run, not worked out on paper.

// precedenceCase carries the rule it stands for, so a failure names the rule that broke
// rather than only the numbers.
type precedenceCase struct {
	source string
	want   string
	rule   string
}

func runPrecedenceCases(t *testing.T, cases []precedenceCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			if got := output(t, tc.source, 0)[0]; got != tc.want {
				t.Errorf("%s printed %s, want %s (%s)", tc.source, got, tc.want, tc.rule)
			}
		})
	}
}

// Each rung against the one below it. A mixed expression has to group the tighter operator
// first, and the two groupings have to give different answers — otherwise the case proves
// nothing. That is why 2 * 3 ^ 2 is here and 2 * 2 ^ 2 is not.
func TestPrecedenceLadder(t *testing.T) {
	runPrecedenceCases(t, []precedenceCase{
		// and over or: (false and true) or true, not false and (true or true).
		{source: "printd false and true or true;", want: "1", rule: "and over or"},
		{source: "printd false and false or true;", want: "1", rule: "and over or"},

		// Comparison over and: (1 equals 1) and (2 equals 2).
		{source: "printd 1 equals 1 and 2 equals 2;", want: "1", rule: "comparison over and"},
		{source: "ident age = 20;\nprintd age bigger 18 and age smaller 65;", want: "1", rule: "comparison over and"},
		{source: "ident age = 70;\nprintd age bigger 18 and age smaller 65;", want: "0", rule: "comparison over and"},

		// Arithmetic over comparison: (2 + 2) equals 4 answers true, where 2 + (2 equals 4)
		// would answer 2; and 20 bigger (4 * 4) answers true, where (20 bigger 4) * 4 would
		// answer 4.
		{source: "printd 2 + 2 equals 4;", want: "1", rule: "arithmetic over comparison"},
		{source: "printd 20 bigger 4 * 4;", want: "1", rule: "arithmetic over comparison"},

		// Multiplication over addition: 2 + (3 * 4).
		{source: "printd 2 + 3 * 4;", want: "14", rule: "* over +"},
		{source: "printd 20 - 3 * 4;", want: "8", rule: "* over -"},

		// Exponentiation over multiplication: 2 * (3 ^ 2).
		{source: "printd 2 * 3 ^ 2;", want: "18", rule: "^ over *"},

		// Unary minus over exponentiation: (-3) ^ 2, which is 9 rather than 2^64 - 9. This
		// is the one rung that reads against the usual convention, where the exponent binds
		// tighter and -3 ^ 2 is -9. A tape is unsigned, so the sign is a wrap either way.
		{source: "printd -3 ^ 2;", want: "9", rule: "unary over ^"},
		// Reading a field binds tighter than any operator, so p.x * p.y multiplies the two
		// fields rather than reading a field of a product.
		{source: "shape Point { x, y };\nident p = Point{3, 4};\nprintd p.x * p.y;", want: "12", rule: ". over *"},
		{source: "shape Point { x, y };\nident p = Point{3, 4};\nprintd p.x + p.y;", want: "7", rule: ". over +"},

		// -(2 * 3) and (-2) * 3 are the same tape, so this one cannot tell the rungs apart —
		// it is here because it is what caught the minus being discarded on the way to the
		// IR, which answered 6.
		{source: "printd -2 * 3;", want: "18446744073709551610", rule: "unary is not discarded"}, // 2^64 - 6
	})
}

// Grouping within one rung. A chain of a single operator is where the two bugs hid: every
// case below answered differently before the fixes, except where the operation is
// associative and the grouping cannot be seen in the result.
func TestAssociativityWithinEachLevel(t *testing.T) {
	runPrecedenceCases(t, []precedenceCase{
		// Division and subtraction group to the left, which is the only grouping that
		// answers what the expression reads as.
		{source: "printd 20 / 5 / 2;", want: "2", rule: "/ groups left"},
		{source: "printd 24 / 2 / 3 / 2;", want: "2", rule: "/ groups left"},
		{source: "printd 4 / 2 * 6;", want: "12", rule: "* and / group left, together"},
		{source: "printd 2 * 3 / 6;", want: "1", rule: "* and / group left, together"},
		{source: "printd 8 / 4 * 2;", want: "4", rule: "* and / group left, together"},
		{source: "printd 20 - 5 - 3;", want: "12", rule: "- groups left"},
		{source: "printd 10 - 3 + 5;", want: "12", rule: "+ and - group left, together"},

		// Exponentiation is the one rung that groups to the right, which is the convention:
		// 2 ^ (2 ^ 3) is 256, where (2 ^ 2) ^ 3 would be 64.
		{source: "printd 2 ^ 2 ^ 3;", want: "256", rule: "^ groups right"},
		{source: "printd 2 ^ 3 ^ 2;", want: "512", rule: "^ groups right"},

		// and and or group left too. Both operations are associative, so a chain answers
		// the same either way — these pin that the chain parses at all and stays true to
		// its operands. The grouping itself is checked in the parser, on the tree.
		{source: "printd true and true and true;", want: "1", rule: "and groups left"},
		{source: "printd true and true and false;", want: "0", rule: "and groups left"},
		{source: "printd false or false or true;", want: "1", rule: "or groups left"},
		{source: "printd false or false or false;", want: "0", rule: "or groups left"},
	})
}

// Parentheses are the way out of every rule above, so each rung has one case that reverses
// it. A precedence bug that also broke parentheses would leave no way to write the
// expression at all.
func TestParenthesesOverrideEveryLevel(t *testing.T) {
	runPrecedenceCases(t, []precedenceCase{
		{source: "printd (2 + 3) * 4;", want: "20", rule: "parentheses beat * over +"},
		{source: "printd (2 * 3) ^ 2;", want: "36", rule: "parentheses beat ^ over *"},
		{source: "printd (2 ^ 2) ^ 3;", want: "64", rule: "parentheses beat right grouping of ^"},
		{source: "printd 20 / (10 / 2);", want: "4", rule: "parentheses beat left grouping of /"},
		{source: "printd 100 - (20 - 5);", want: "85", rule: "parentheses beat left grouping of -"},
		{source: "printd false and (true or true);", want: "0", rule: "parentheses beat and over or"},
		{source: "printd -(2 + 3);", want: "18446744073709551611", rule: "parentheses gather what is negated"}, // 2^64 - 5
	})
}

// A chain of comparisons groups to the right: 3 bigger 2 bigger 1 is 3 bigger (2 bigger 1),
// which compares 3 against the truth value 1 and answers true. Grouping to the left would
// compare (3 bigger 2), which is 1, against 1 and answer false.
//
// This is the last rung recursing to the right that is not meant to — ParseRelExpr has the
// shape ParseMultExpr and ParseBoolExpr had before they were fixed. It is written down here
// so the behaviour is not a surprise, and this is the test that has to change on the day
// ParseRelExpr is fixed.
func TestComparisonChainsStillGroupToTheRight(t *testing.T) {
	runPrecedenceCases(t, []precedenceCase{
		{source: "printd 3 bigger 2 bigger 1;", want: "1", rule: "comparison groups right, for now"},
		{source: "printd 1 smaller 2 smaller 3;", want: "0", rule: "comparison groups right, for now"},
	})
}
