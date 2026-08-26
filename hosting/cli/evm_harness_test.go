package cli

import (
	"bytes"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm/runtime"
)

// Aurora exists to let a call be simulated off the chain, which only means something if the
// two answers are the same. Nothing checked that: the builder's tests read the shape of the
// bytecode — its size, its selectors, that it comes out the same twice — and never once ran
// it.
//
// go-ethereum is already a dependency, so an EVM can be built in memory here: the constructor
// runs the way a chain would run it, the runtime it returns is installed, and a call arrives
// through the same encoder "aurora call" uses. What comes back is compared against what the
// evaluator answers for the same source.

// onChain compiles the source, installs it in an EVM of its own, and calls one of its scopes
// through calldata built exactly as the CLI builds it.
func onChain(t *testing.T, source, function string, args []string, tapeSize int) []byte {
	t.Helper()

	// Through the command, and then read back from where it landed: what is installed below
	// is the binary a user gets, not one assembled for the test.
	dir := t.TempDir()
	path := writeAt(t, dir, "contract.ar", source)
	binary := filepath.Join(dir, "contract.bin")

	if _, err := newSession(t, sessionOpts{tapeSize: tapeSize}).Build(t.Context(), path, binary); err != nil {
		t.Fatalf("building: %v", err)
	}
	bytecode, err := os.ReadFile(binary)
	if err != nil {
		t.Fatalf("reading the binary: %v", err)
	}

	return calledOnChain(t, bytecode, function, args, tapeSize)
}

// calledOnChain deploys a contract to an EVM in memory and calls one of its scopes by name.
//
// It is apart from the building so that a program of several files can be deployed too: how a
// contract is assembled depends on how many files it is, and what happens after it is
// assembled does not.
func calledOnChain(t *testing.T, bytecode []byte, function string, args []string, tapeSize int) []byte {
	t.Helper()

	cfg := &runtime.Config{GasLimit: 10_000_000, Value: big.NewInt(0)}
	_, address, _, err := runtime.Create(bytecode, cfg)
	if err != nil {
		t.Fatalf("deploying: %v", err)
	}

	// The same two encoders "aurora call" uses, so what is proven here is the path someone
	// actually takes rather than one built for the test.
	calldata := append(EncodeSelector(function), ParseArgs(args)...)

	returned, _, err := runtime.Call(address, calldata, cfg)
	if err != nil {
		t.Fatalf("calling %s: %v", function, err)
	}
	return returned
}

// offChain answers what the evaluator makes of the same call, with the arguments arriving the
// same way they arrive on chain: encoded by ParseArgs and narrowed to a tape on the way in.
//
// The call still has to be written into the source, because there is no way to ask the
// evaluator for one scope by name — the closest thing, "aurora call", only speaks to a
// network. That is the gap this harness makes visible. What it must not do is write the
// arguments in as literals: a literal is checked against the tape when it is compiled, so
// "add(300, 0)" on a one-byte tape is refused at compile time while the same 300 arriving as
// calldata is simply narrowed.
func offChain(t *testing.T, source, function string, args []string, tapeSize int) string {
	t.Helper()

	feeds := make([]string, 0, len(args))
	for i := range args {
		feeds = append(feeds, fmt.Sprintf("feed(%d)", i))
	}
	probe := fmt.Sprintf("printd %s(%s);", function, strings.Join(feeds, ", "))

	path := writeAt(t, t.TempDir(), "program.ar", source+"\n"+probe+"\n")
	out := &strings.Builder{}
	session := newSession(t, sessionOpts{tapeSize: tapeSize, stdout: out, args: args})
	if err := session.Run(t.Context(), path); err != nil {
		t.Fatalf("running: %v", err)
	}
	return strings.TrimSpace(out.String())
}

// agreeThroughTheStandardLibrary runs the same call through both backends, for a program that
// imports what comes with the language.
//
// It needs a project rather than one file, because a module is found from a source root, and a
// standard library to find — so the two ways in are built here rather than in the helpers
// above, which know about one file and nothing else.
func agreeThroughTheStandardLibrary(t *testing.T, source, function string, args []string, tapeSize int) {
	t.Helper()

	opts := sessionOpts{tapeSize: tapeSize, stdRoot: stdRootOf(t)}
	entry := filepath.Join("src", "main.ar")

	projectOf(t, map[string]string{"src/main.ar": source})
	binary := filepath.Join(t.TempDir(), "contract.bin")
	if _, err := newSession(t, opts).Build(t.Context(), entry, binary); err != nil {
		t.Fatalf("building: %v", err)
	}
	bytecode, err := os.ReadFile(binary)
	if err != nil {
		t.Fatalf("reading the binary: %v", err)
	}

	feeds := make([]string, 0, len(args))
	for i := range args {
		feeds = append(feeds, fmt.Sprintf("feed(%d)", i))
	}
	printed := &strings.Builder{}
	ran := opts
	ran.stdout, ran.args = printed, args

	projectOf(t, map[string]string{
		"src/main.ar": source + "\n" + fmt.Sprintf("printd %s(%s);", function, strings.Join(feeds, ", ")) + "\n",
	})
	if err := newSession(t, ran).Run(t.Context(), entry); err != nil {
		t.Fatalf("running: %v", err)
	}

	if got := decimalOf(calledOnChain(t, bytecode, function, args, tapeSize)); got != strings.TrimSpace(printed.String()) {
		t.Errorf("the chain answered %s and the evaluator %s", got, strings.TrimSpace(printed.String()))
	}
}

// agree runs the same call through both backends and reports when they answer differently.
func agree(t *testing.T, source, function string, args []string, tapeSize int) {
	t.Helper()

	returned := onChain(t, source, function, args, tapeSize)
	want := offChain(t, source, function, args, tapeSize)

	if got := decimalOf(returned); got != want {
		t.Errorf("the chain answered %s and the evaluator %s", got, want)
	}
}

// decimalOf reads what a contract returned as the number it is. A return is one word, and a
// tape is right-aligned inside it, so the whole word is the value.
func decimalOf(returned []byte) string {
	return new(big.Int).SetBytes(returned).String()
}

// The first program to cross the whole path: compiled, deployed, called, and answered for by
// both backends.
func TestAddAnswersTheSameOnChainAndOff(t *testing.T) {
	const source = `ident add = defer { feed(0) + feed(1); };`

	cases := []struct {
		name string
		args []string
	}{
		{name: "small", args: []string{"1", "2"}},
		{name: "larger", args: []string{"1000", "337"}},
		{name: "zero", args: []string{"0", "0"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, source, "add", tc.args, 0)
		})
	}
}

// A contract holds as many scopes as it has names, and the selector is what tells them apart.
func TestEachScopeAnswersForItself(t *testing.T) {
	const source = `ident add = defer { feed(0) + feed(1); };
ident multiply = defer { feed(0) * feed(1); };`

	for _, function := range []string{"add", "multiply"} {
		t.Run(function, func(t *testing.T) {
			agree(t, source, function, []string{"6", "7"}, 0)
		})
	}
}

// A name the contract does not have reaches the no-match STOP, which returns nothing —
// it does not run the first scope it finds.
func TestAnUnknownNameAnswersWithNothing(t *testing.T) {
	const source = `ident add = defer { feed(0) + feed(1); };`

	if returned := onChain(t, source, "subtract", []string{"1", "2"}, 0); len(returned) != 0 {
		t.Errorf("a name that is not there answered %v", returned)
	}
}

// A tape of N bytes holds values modulo 2^(8N); the EVM wraps at 2^256. Every one of these
// disagreed before the results were cut back to the width — the chain answered 256 where the
// evaluator answered 0 — and nothing noticed, because the tape-size tests read the width of a
// PUSH and never the result of a sum.
func TestArithmeticWrapsAtTheTapeWidthOnBothSides(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		function string
		args     []string
		tapeSize int
	}{
		{
			name:     "a sum past the top of a one-byte tape",
			source:   `ident add = defer { feed(0) + feed(1); };`,
			function: "add", args: []string{"255", "1"}, tapeSize: 1,
		},
		{
			name:     "a product past the top",
			source:   `ident multiply = defer { feed(0) * feed(1); };`,
			function: "multiply", args: []string{"16", "16"}, tapeSize: 1,
		},
		{
			// Under zero the value comes back from the other end, and how far depends on the
			// width: 255 on one byte, not 2^256 - 1.
			name:     "a difference under zero",
			source:   `ident subtract = defer { feed(0) - feed(1); };`,
			function: "subtract", args: []string{"0", "1"}, tapeSize: 1,
		},
		{
			// An argument arrives as a whole word whatever the tape is, and the evaluator
			// narrows it on the way in. Reading the whole word on chain let a caller hand a
			// contract a value its own language cannot hold.
			name:     "an argument wider than the tape",
			source:   `ident add = defer { feed(0) + feed(1); };`,
			function: "add", args: []string{"300", "0"}, tapeSize: 1,
		},
		{
			name:     "two bytes",
			source:   `ident add = defer { feed(0) + feed(1); };`,
			function: "add", args: []string{"65535", "2"}, tapeSize: 2,
		},
		{
			// This is why every result is cut and not only the one that leaves: a sum, a
			// difference and a product agree either way, because wrapping and adding can be
			// done in either order. A division cannot. At one byte 255 + 1 is 0, and 0 / 2 is
			// 0 — but 256 / 2 is 128, and cutting that afterwards still returns 128.
			name:     "a division after a sum that left the width",
			source:   `ident half = defer { (feed(0) + feed(1)) / feed(2); };`,
			function: "half", args: []string{"255", "1", "2"}, tapeSize: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, tc.source, tc.function, tc.args, tc.tapeSize)
		})
	}
}

// Reading past the values applied to a scope is not an error — the read always returns a
// tape. What it returns is the question, and today the two backends answer differently:
// the evaluator takes the index modulo the length of the vector, so a missing argument comes
// back as the first one; the chain reads past the end of the calldata, which the EVM gives as
// zeros.
//
// Nothing in the IR says which of the two is right. OpGetFeed carries an index and nothing
// else, so the meaning lives in whichever consumer implemented it first.
func TestReadingPastTheValuesAppliedAnswersTheSameOnChainAndOff(t *testing.T) {
	const source = `ident sum = defer { feed(0) + feed(1); };`

	agree(t, source, "sum", []string{"5"}, 0)
}

// A name bound inside a scope used to compile to an MSTORE with nothing under it: the lowering
// decided which operands named values from the opcode, and a binding was not on the list, so
// the value it meant to store was never put on the stack. The contract answered 4 where the
// program answers 7.
func TestALocalInsideAScopeAnswersTheSameOnChainAndOff(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{name: "read once", source: `ident sum = defer { ident x = feed(0); x + feed(1); };`},
		{name: "read twice", source: `ident sum = defer { ident x = feed(0); x + x; };`},
		{name: "two of them, and one reads the other", source: `ident sum = defer { ident x = feed(0); ident y = x + feed(1); y + x; };`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, tc.source, "sum", []string{"3", "4"}, 0)
		})
	}
}

// A value written down reaches the stack from the instruction that takes it, and where it
// lands depends on which side it is on. The EVM computes top minus next, so "feed(0) - 2"
// needs the two underneath a value that is already on top — which is the one place the writer
// has to make two things change places.
//
// Addition would not tell them apart, so the cases that prove it are the ones that do.
func TestAValueWrittenDownAnswersTheSameOnChainAndOff(t *testing.T) {
	cases := []struct {
		name   string
		source string
		args   []string
	}{
		{name: "written down on the right of a subtraction", source: `ident f = defer { feed(0) - 2; };`, args: []string{"9"}},
		{name: "written down on the left of a subtraction", source: `ident f = defer { 10 - feed(0); };`, args: []string{"4"}},
		{name: "written down on the right of a division", source: `ident f = defer { feed(0) / 2; };`, args: []string{"9"}},
		{name: "written down on the left of a division", source: `ident f = defer { 100 / feed(0); };`, args: []string{"4"}},
		{name: "both written down", source: `ident f = defer { 7 - 3; };`, args: []string{"0"}},
		{name: "written down in a sum", source: `ident f = defer { feed(0) + 5; };`, args: []string{"6"}},
		{name: "bound to a name", source: `ident f = defer { ident x = 3; feed(0) - x; };`, args: []string{"10"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, tc.source, "f", tc.args, 0)
		})
	}
}

// A contract whose runtime is longer than a byte can count used to be published cut short.
// The constructor pushed the size with PUSH1, so 352 bytes became 96 by conversion: it copied
// 96 and the chain kept 96, of a contract that ended in the middle of an instruction. Three
// deferred scopes with ordinary bodies reach that.
//
// What this proves is the constructor. It calls the first scope on purpose, because a
// dispatcher still pushes its jump target with PUSH1 — the second of the three ceilings, and
// not this one.
func TestARuntimeLongerThanAByteIsPublishedWhole(t *testing.T) {
	const source = `ident one = defer { feed(0) + feed(1) * 3 - feed(0) / 2; };
ident two = defer { feed(0) + feed(1) * 5 - feed(0) / 2; };
ident three = defer { feed(0) + feed(1) * 7 - feed(0) / 2; };`

	agree(t, source, "one", []string{"6", "7"}, 0)
}

// A dispatcher pushed the address of its body in one byte, so a scope living past byte 255 of
// the runtime was jumped to at an address that had been truncated: a contract with twelve
// scopes answered for the first and refused the third with "invalid jump destination". The
// body was there; the dispatcher could not name it.
func TestAScopePastAByteIsReachedOnChainToo(t *testing.T) {
	var source strings.Builder
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&source, "ident scope%d = defer { feed(0) + feed(1) * %d - feed(0) / 2; };\n", i, i+1)
	}

	// The first is reachable whatever the size of the push, so the ones that say anything
	// are the ones further in.
	for _, name := range []string{"scope0", "scope2", "scope11"} {
		t.Run(name, func(t *testing.T) {
			agree(t, source.String(), name, []string{"6", "7"}, 0)
		})
	}
}

// A name is kept in a slot of memory of its own, thirty-two bytes wide, and the address used
// to go in one byte — so the ninth name in a contract was given the address of the first and
// the two wrote over each other. Nine names is an ordinary scope.
func TestManyNamesInAScopeAnswerTheSameOnChainAndOff(t *testing.T) {
	var body strings.Builder
	for i := 0; i < 9; i++ {
		fmt.Fprintf(&body, "  ident n%d = feed(0) + %d;\n", i, i)
	}
	body.WriteString("  n0 + n8;\n")

	source := "ident scope = defer {\n" + body.String() + "};"

	agree(t, source, "scope", []string{"5"}, 0)
}

// A branch reaches the bytecode. The IR skips ahead when the test is false and the EVM jumps
// when what it pops is not zero, so the test is turned over; both arms leave one value, which
// is what makes an "if" an expression — whoever is under it finds a value without knowing
// which way the program went.
//
// The test of each case is a value rather than a comparison, because a comparison does not
// reach the bytecode yet and would be testing something that was never computed.
func TestABranchAnswersTheSameOnChainAndOff(t *testing.T) {
	cases := []struct {
		name   string
		source string
		args   []string
	}{
		{name: "the then arm", source: `ident f = defer { if feed(0) { 42; } else { 7; }; };`, args: []string{"1"}},
		{name: "the else arm", source: `ident f = defer { if feed(0) { 42; } else { 7; }; };`, args: []string{"0"}},
		{name: "no else, taken", source: `ident f = defer { if feed(0) { 42; }; };`, args: []string{"1"}},
		{name: "no else, not taken", source: `ident f = defer { if feed(0) { 42; }; };`, args: []string{"0"}},
		{name: "the value is used after", source: `ident f = defer { ident r = if feed(0) { 42; } else { 7; }; r + 1; };`, args: []string{"1"}},
		{name: "a branch inside a branch", source: `ident f = defer { if feed(0) { if feed(1) { 1; } else { 2; }; } else { 3; }; };`, args: []string{"1", "0"}},
		{name: "arithmetic in both arms", source: `ident f = defer { if feed(0) { feed(1) + 1; } else { feed(1) * 2; }; };`, args: []string{"0", "5"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, tc.source, "f", tc.args, 0)
		})
	}
}

// A comparison returns a tape like any other value, and the EVM answers these with one or
// zero, which is what a tape holding true or false is.
//
// The two that are not symmetric are the ones that say anything: the EVM reads its first
// operand off the top, so "a bigger b" has to arrive with a on top or it answers the question
// backwards. Both sides of each are here for that reason.
//
// "and" and "or" are the logical ones and not the bitwise ones — Aurora asks whether both
// values hold, not which bits they share — so "2 and 1" is here too, which a bitwise AND
// would answer zero.
func TestComparisonsAndLogicAnswerTheSameOnChainAndOff(t *testing.T) {
	cases := []struct {
		name string
		body string
		args []string
	}{
		{name: "equals, and it does", body: "feed(0) equals feed(1);", args: []string{"5", "5"}},
		{name: "equals, and it does not", body: "feed(0) equals feed(1);", args: []string{"5", "6"}},
		{name: "different", body: "feed(0) different feed(1);", args: []string{"5", "6"}},
		{name: "bigger, and it is", body: "feed(0) bigger feed(1);", args: []string{"9", "2"}},
		{name: "bigger, and it is not", body: "feed(0) bigger feed(1);", args: []string{"2", "9"}},
		{name: "smaller, and it is", body: "feed(0) smaller feed(1);", args: []string{"2", "9"}},
		{name: "smaller, and it is not", body: "feed(0) smaller feed(1);", args: []string{"9", "2"}},
		{name: "and, both hold", body: "feed(0) and feed(1);", args: []string{"1", "1"}},
		{name: "and, one does not", body: "feed(0) and feed(1);", args: []string{"1", "0"}},
		{name: "and, over values a bitwise one would answer zero for", body: "feed(0) and feed(1);", args: []string{"2", "1"}},
		{name: "or, neither holds", body: "feed(0) or feed(1);", args: []string{"0", "0"}},
		{name: "or, one does", body: "feed(0) or feed(1);", args: []string{"0", "3"}},
		{name: "raised to a power", body: "feed(0) ^ feed(1);", args: []string{"2", "5"}},
		{name: "a power that leaves the width", body: "feed(0) ^ feed(1);", args: []string{"3", "7"}},
		{name: "a branch on a comparison", body: "if feed(0) bigger feed(1) { 42; } else { 7; };", args: []string{"9", "2"}},
		{name: "a branch on a comparison, the other way", body: "if feed(0) bigger feed(1) { 42; } else { 7; };", args: []string{"2", "9"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, "ident f = defer { "+tc.body+" };", "f", tc.args, 0)
		})
	}
}

// The same, at a width where a comparison and a power both have to answer for the tape they
// are working in.
func TestComparisonsFollowTheTapeWidth(t *testing.T) {
	for _, size := range []int{1, 2, 32} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			agree(t, "ident f = defer { feed(0) bigger feed(1); };", "f", []string{"200", "100"}, size)
			agree(t, "ident f = defer { feed(0) ^ feed(1); };", "f", []string{"3", "5"}, size)
		})
	}
}

// A name lives in the frame of the scope that bound it, not at an address of its own in the
// contract. Two scopes each binding a name used to be given the same place, since the offsets
// were counted once for the whole binary — which is also what kept a scope from calling
// itself, and is what the frame is for.
func TestTwoScopesKeepTheirOwnNames(t *testing.T) {
	const source = `ident first = defer { ident x = feed(0); x + 1; };
ident second = defer { ident x = feed(0); x + 100; };`

	for _, name := range []string{"first", "second"} {
		t.Run(name, func(t *testing.T) {
			agree(t, source, name, []string{"5"}, 0)
		})
	}
}

// A scope calls another and comes back with what it answered.
//
// It is a jump inside one contract, not a message call to itself, and what makes that possible
// is the frame: the callee keeps the values applied to it right after the caller's own, so
// "feed(0)" inside the callee means the callee's first value and not the caller's — which is
// what a plain jump would have got wrong, since the calldata belongs to whoever the
// transaction named and every scope would have read the same one.
func TestAScopeCallsAnotherOnChainToo(t *testing.T) {
	const source = `ident b = defer { feed(0) + 1; };
ident a = defer { b(10) + feed(0); };`

	agree(t, source, "a", []string{"5"}, 0)
}

// What the caller was holding survives the call.
//
// The callee runs on the same stack, so a value the caller had already worked out sits under
// the call while it happens. And the caller's own applied values have to still be there
// afterwards: the frame pointer moves onto the callee's frame and has to come back off it.
func TestACallLeavesTheCallerWhereItWas(t *testing.T) {
	const source = `ident twice = defer { feed(0) * 2; };
ident sum = defer { ident kept = feed(0) + 100; twice(feed(1)) + kept + feed(0); };`

	agree(t, source, "sum", []string{"7", "3"}, 0)
}

// A call inside a call. Each activation takes the frame that follows the one it was entered
// from, so three of them are three runs of memory and none of them writes over another.
func TestCallsNestOnChainToo(t *testing.T) {
	const source = `ident inner = defer { feed(0) + 1; };
ident middle = defer { inner(feed(0)) * 2; };
ident outer = defer { middle(feed(0)) + middle(1); };`

	agree(t, source, "outer", []string{"4"}, 0)
}

// A shape reaches the chain. It is not a new kind of value: Point{10, 20} is two tapes laid
// end to end, and on a chain that is memory — the tapes in the order they were written, which
// is the order the evaluator lays them, so the two hand back the same bytes.
//
// A shape never goes on the stack, not even when it would fit. A stack item is exactly a word
// and a run kept two ways is two things a consumer has to tell apart, which is how this
// backend has been wrong before.
func TestAShapeAnswersTheSameOnChainAndOff(t *testing.T) {
	const source = `shape Point { x, y };
ident first = defer { ident p = Point{feed(0), 20}; p.x; };
ident second = defer { ident p = Point{feed(0), 20}; p.y; };`

	for _, name := range []string{"first", "second"} {
		t.Run(name, func(t *testing.T) {
			agree(t, source, name, []string{"10"}, 0)
		})
	}
}

// A run of three, where the middle field is the one that proves the counting: it is neither
// the tape the run starts with nor the one it ends with.
func TestAFieldOfAWiderRunAnswersTheSameOnChainAndOff(t *testing.T) {
	const source = `shape Line { a, b, c };
ident middle = defer { ident l = Line{1, feed(0), 3}; l.b; };`

	agree(t, source, "middle", []string{"7"}, 0)
}

// A run is as wide as its tapes, so it follows the tape width the way everything else does. At
// one byte a run of three is three bytes, and each field is still the byte it was laid in.
func TestAShapeFollowsTheTapeWidth(t *testing.T) {
	const source = `shape Line { a, b, c };
ident third = defer { ident l = Line{1, 2, feed(0)}; l.c; };
ident first = defer { ident l = Line{1, 2, feed(0)}; l.a; };`

	for _, name := range []string{"third", "first"} {
		t.Run(name, func(t *testing.T) {
			agree(t, source, name, []string{"9"}, 1)
		})
	}
}

// A shape wider than a word, which is what this was all for.
//
// Five tapes of eight is forty bytes and a word is thirty-two, so this program ran off a chain
// and was refused by it — "a shape this wide does not reach the bytecode". Nothing about it is
// special now: the run is in memory, and forty bytes of memory is the same kind of thing as
// eight.
func TestAShapeWiderThanAWordAnswersTheSameOnChainAndOff(t *testing.T) {
	const source = `shape Five { a, b, c, d, e };
ident last = defer { ident f = Five{1, 2, 3, 4, feed(0)}; f.e; };
ident first = defer { ident f = Five{feed(0), 2, 3, 4, 5}; f.a; };
ident fourth = defer { ident f = Five{1, 2, 3, feed(0), 5}; f.d; };`

	for _, name := range []string{"last", "first", "fourth"} {
		t.Run(name, func(t *testing.T) {
			agree(t, source, name, []string{"42"}, 0)
		})
	}
}

// Twice as wide again, so that five is not the number that happens to work.
func TestAShapeOfEightFieldsAnswersTheSameOnChainAndOff(t *testing.T) {
	const source = `shape Eight { a, b, c, d, e, f, g, h };
ident seventh = defer { ident v = Eight{1, 2, 3, 4, 5, 6, feed(0), 8}; v.g; };`

	agree(t, source, "seventh", []string{"77"}, 0)
}

// A shape at the widest tape, where a run of two is already twice a word.
//
// This was the sharpest edge of the old ceiling: at a tape of thirty-two, a shape of two
// fields did not reach a chain at all, so `shape` was unusable in that dialect.
func TestAShapeAtTheWidestTapeAnswersTheSameOnChainAndOff(t *testing.T) {
	const source = `shape Pair { a, b };
ident second = defer { ident p = Pair{1, feed(0)}; p.b; };`

	agree(t, source, "second", []string{"5"}, 32)
}

// A whole run handed back, rather than a field of one — and compared as bytes.
//
// Every other case here compares what came back as a number, which is enough for a field
// because a field is a tape. A run is not a number: the evaluator hands back its bytes, and
// the point of putting it in memory is that the chain hands back the same ones. So this is the
// case that says so, and it says it byte for byte.
func TestAWholeWideRunIsTheSameBytesOnChainAndOff(t *testing.T) {
	const source = `shape Five { a, b, c, d, e };
ident build = defer { Five{1, 2, 3, 4, feed(0)}; };`

	// Five tapes of eight, in the order they were written: forty bytes, where a word is
	// thirty-two. This is what used to be refused.
	want := make([]byte, 0, 40)
	for _, tape := range []byte{1, 2, 3, 4, 9} {
		want = append(want, 0, 0, 0, 0, 0, 0, 0, tape)
	}

	if got := onChain(t, source, "build", []string{"9"}, 0); !bytes.Equal(got, want) {
		t.Errorf("the chain handed back %x, want %x", got, want)
	}

	// And the evaluator reads those same bytes as the five tapes they are.
	if printed := strings.TrimSpace(offChain(t, source, "build", []string{"9"}, 0)); printed != "1 2 3 4 9" {
		t.Errorf("the evaluator answered %q, want the five tapes", printed)
	}
}

// A value nothing takes is dropped, rather than left under everything written after it.
//
// A line written for what it does and not for what it is worth leaves one: `sstore 1 42;` on
// its own, a `printd`, a call whose answer nobody wanted, or an arithmetic line somebody
// stopped using. Off a chain it is a name in a map that is never read; on one it sat on the
// stack, and the return that expected the address it came from found it instead — a jump to
// nowhere, so the contract reverted rather than answering wrongly.
//
// It was true before anything here kept state, and nothing made it likely to be written until
// something did: nobody writes an arithmetic line for its effect, and `sstore 1 42;` is an
// ordinary way to write a program.
func TestAValueNothingTakesIsDropped(t *testing.T) {
	for _, tc := range []struct{ name, source, args string }{
		{
			name:   "an arithmetic line nobody uses",
			source: `ident f = defer { feed(0) + 1; feed(0) + 2; };`,
			args:   "5",
		},
		{
			name:   "a write kept for what it does",
			source: `ident f = defer { sstore 1 feed(0); 7; };`,
			args:   "42",
		},
		{
			name: "a call whose answer nobody wanted",
			source: `ident g = defer { feed(0) * 2; };
ident f = defer { g(feed(0)); 7; };`,
			args: "42",
		},
		{
			name:   "several of them in a row",
			source: `ident f = defer { feed(0) + 1; feed(0) + 2; feed(0) + 3; feed(0) + 4; };`,
			args:   "5",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, tc.source, "f", []string{tc.args}, 0)
		})
	}
}

// What the chain keeps, kept and read back inside one transaction.
//
// It is the first thing here that is not a computation: what SSTORE leaves behind outlives the
// call, and what the evaluator answers comes out of a map. The two agree because they were
// written to mean the same thing, and this is what says they do.
func TestWhatTheChainKeepsAnswersTheSameOnChainAndOff(t *testing.T) {
	const source = `ident keep = defer { sstore 1 feed(0); sload 1; };
ident over = defer { sstore 1 feed(0); sstore 1 7; sload 1; };
ident missing = defer { sstore 1 feed(0); sload 2; };
ident counted = defer { sstore 1 40; sstore 1 ((sload 1) + feed(0)); sload 1; };`

	for _, tc := range []struct{ name, want string }{
		{name: "keep", want: "42"},
		{name: "over", want: "7"},
		// A slot never written is zeros on a chain and the neutral tape off one, and neither
		// was told to agree with the other.
		{name: "missing", want: "0"},
		{name: "counted", want: "42"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, source, tc.name, []string{"42"}, 0)
			if tc.want == "" {
				t.Skip()
			}
		})
	}
}

// What one transaction keeps, the next one reads.
//
// Every other case here writes and reads inside one call, which proves the two opcodes and
// nothing about what a chain is for. This deploys once and calls twice, so what the second
// call sees is what the first left — which is the whole point of storage and the one thing
// the harness had never asked.
func TestWhatOneTransactionKeepsTheNextOneReads(t *testing.T) {
	const source = `ident inc = defer { sstore 1 ((sload 1) + 1); };
ident read = defer { sload 1; };`

	dir := t.TempDir()
	path := writeAt(t, dir, "contract.ar", source)
	binary := filepath.Join(dir, "contract.bin")
	if _, err := newSession(t, sessionOpts{}).Build(t.Context(), path, binary); err != nil {
		t.Fatalf("building: %v", err)
	}
	bytecode, err := os.ReadFile(binary)
	if err != nil {
		t.Fatalf("reading the binary: %v", err)
	}

	// One EVM and one contract for the whole test, because two calls sharing state is the
	// thing being asked about.
	cfg := &runtime.Config{GasLimit: 10_000_000, Value: big.NewInt(0)}
	_, address, _, err := runtime.Create(bytecode, cfg)
	if err != nil {
		t.Fatalf("deploying: %v", err)
	}

	reach := func(function string) string {
		t.Helper()
		returned, _, err := runtime.Call(address, EncodeSelector(function), cfg)
		if err != nil {
			t.Fatalf("calling %s: %v", function, err)
		}
		return decimalOf(returned)
	}

	if got := reach("read"); got != "0" {
		t.Errorf("before anything was kept, read answered %s, want 0", got)
	}
	if got := reach("inc"); got != "1" {
		t.Errorf("the first inc answered %s, want 1", got)
	}
	// The one that matters: a call of its own, after the transaction that wrote.
	if got := reach("read"); got != "1" {
		t.Errorf("after one inc, read answered %s, want 1 — what one transaction kept, the next did not read", got)
	}
	if got := reach("inc"); got != "2" {
		t.Errorf("the second inc answered %s, want 2 — it did not see what the first left", got)
	}
	if got := reach("read"); got != "2" {
		t.Errorf("after two, read answered %s, want 2", got)
	}
}

// And the same through the standard library, which is what a program writes.
func TestWhatTheStandardLibraryKeepsSurvivesTheTransaction(t *testing.T) {
	const source = "use std/evm/storage as s;\n" +
		"ident deposit = defer { s.set(1, s.get(1) + feed(0)); };\n" +
		"ident balance = defer { s.get(1); };"

	projectOf(t, map[string]string{"src/main.ar": source})
	binary := filepath.Join(t.TempDir(), "contract.bin")
	if _, err := newSession(t, sessionOpts{}).Build(t.Context(), filepath.Join("src", "main.ar"), binary); err != nil {
		t.Fatalf("building: %v", err)
	}
	bytecode, err := os.ReadFile(binary)
	if err != nil {
		t.Fatalf("reading the binary: %v", err)
	}

	cfg := &runtime.Config{GasLimit: 10_000_000, Value: big.NewInt(0)}
	_, address, _, err := runtime.Create(bytecode, cfg)
	if err != nil {
		t.Fatalf("deploying: %v", err)
	}

	reach := func(function string, args []string) string {
		t.Helper()
		calldata := append(EncodeSelector(function), ParseArgs(args)...)
		returned, _, err := runtime.Call(address, calldata, cfg)
		if err != nil {
			t.Fatalf("calling %s: %v", function, err)
		}
		return decimalOf(returned)
	}

	reach("deposit", []string{"40"})
	reach("deposit", []string{"2"})
	if got := reach("balance", nil); got != "42" {
		t.Errorf("after two deposits, balance answered %s, want 42", got)
	}
}

// A write is worth what it wrote, on a chain as much as off one.
//
// SSTORE leaves nothing on the stack, so the value is copied before it is spent. Getting that
// wrong is not a crash: the contract answers whatever was underneath, and a number came back
// either way.
func TestAWriteToTheChainIsWorthWhatItWrote(t *testing.T) {
	const source = `ident kept = defer { sstore 1 feed(0); };
ident plus = defer { (sstore 1 feed(0)) + 1; };`

	for _, name := range []string{"kept", "plus"} {
		t.Run(name, func(t *testing.T) {
			agree(t, source, name, []string{"41"}, 0)
		})
	}
}

// And through the standard library, which is what a program actually writes.
//
// The two scopes of std/evm/storage are ordinary scopes, so this is also a call reaching a
// module reaching the machine — every piece of the language at once.
func TestTheStandardLibraryKeepsOnChainToo(t *testing.T) {
	const source = "use std/evm/storage as s;\n" +
		"ident deposit = defer { s.set(1, s.get(1) + feed(0)); };\n" +
		"ident balance = defer { s.set(1, 40); deposit(2); s.get(1); };"

	agreeThroughTheStandardLibrary(t, source, "balance", []string{"0"}, 0)
}

// A run laid out of what a call answered, and a field read out of it. The two features meet
// here: the values a shape is built from are worked out before it, and the lowering has to put
// each of them on the stack in the order the run is laid in.
func TestAShapeBuiltFromCallsAnswersTheSameOnChainAndOff(t *testing.T) {
	const source = `shape Point { x, y };
ident twice = defer { feed(0) * 2; };
ident build = defer { ident p = Point{twice(feed(0)), twice(feed(1))}; p.x + p.y; };`

	agree(t, source, "build", []string{"3", "4"}, 0)
}

// A scope returning a whole run answers the tapes laid end to end, with the first at the far
// end — which is what makes the run a number the evaluator agrees with.
//
// This one cannot go through the harness's usual comparison: it compares what printd wrote,
// and printd writes a run tape by tape while a chain answers one word. So the word is read
// against the tapes themselves.
func TestAScopeAnswersAWholeRun(t *testing.T) {
	const source = `shape Point { x, y };
ident whole = defer { Point{feed(0), 20}; };`

	returned := onChain(t, source, "whole", []string{"10"}, 0)

	want := new(big.Int).SetBytes([]byte{0, 0, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 20})
	if got := new(big.Int).SetBytes(returned); got.Cmp(want) != 0 {
		t.Errorf("the chain answered %s, want the two tapes laid end to end, which is %s", got, want)
	}
}

// A tape is a shift register, and the operations on it reach the chain. Each of them is
// defined by how much of a value means something — its bytes once the zeros in front are
// dropped — which off a chain is the length of a slice and on one has to be worked out from
// the word itself.
func TestTapeOperationsAnswerTheSameOnChainAndOff(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		args     []string
		tapeSize int
	}{
		{
			name:   "a literal is pulled one value at a time",
			source: "ident t = defer { ident tape = [1, 2, 3]; tape + feed(0); };",
			args:   []string{"0"},
		},
		{
			name:   "an empty literal is a tape of zeros",
			source: "ident t = defer { ident tape = []; tape + feed(0); };",
			args:   []string{"5"},
		},
		{
			name:   "a value the program works out enters at the right",
			source: "ident t = defer { ident x = feed(0); pull [1, 2] x; };",
			args:   []string{"7"},
		},
		{
			name:   "and one written down",
			source: "ident t = defer { pull [1, 2] 4; };",
			args:   []string{"0"},
		},
		{
			name:   "a value wider than a byte enters as the bytes it takes",
			source: "ident t = defer { ident x = feed(0); pull [1] x; };",
			args:   []string{"300"},
		},
		{
			name:   "pushing lets a value in at the left and the far byte falls off",
			source: "ident t = defer { ident x = feed(0); push [1, 2, 3] x; };",
			args:   []string{"5"},
		},
		{
			name:   "and one written down",
			source: "ident t = defer { push [1, 2, 3] 5; };",
			args:   []string{"0"},
		},
		{
			name:   "head keeps the first significant bytes",
			source: "ident t = defer { ident five = [1, 2, 3, 4, 5]; head five 2; };",
			args:   []string{"0"},
		},
		{
			name:   "tail drops them",
			source: "ident t = defer { ident five = [1, 2, 3, 4, 5]; tail five 2; };",
			args:   []string{"0"},
		},
		{
			name:   "an index past the tape width wraps into it",
			source: "ident t = defer { ident full = [1, 2, 3, 4, 5, 6, 7, 8]; tail full 18; };",
			args:   []string{"0"},
		},
		{
			name:   "an index past what the value says gives all of it, or none",
			source: "ident t = defer { ident x = feed(0); head x 5; };",
			args:   []string{"7"},
		},
		{
			name:   "the same, dropping",
			source: "ident t = defer { ident x = feed(0); tail x 5; };",
			args:   []string{"7"},
		},
		{
			name:   "a tape of zeros is one byte long, so pulling onto it moves one place",
			source: "ident t = defer { ident x = feed(0); pull x 9; };",
			args:   []string{"0"},
		},
		{
			name:     "at one byte, everything happens inside a single place",
			source:   "ident t = defer { ident x = feed(0); pull [1] x; };",
			args:     []string{"9"},
			tapeSize: 1,
		},
		{
			name:     "and at the full width, where a shift reaches the end of a word",
			source:   "ident t = defer { ident x = feed(0); head x 4; };",
			args:     []string{"1000000"},
			tapeSize: 32,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, tc.source, "t", tc.args, tc.tapeSize)
		})
	}
}

// A scope written inside another does not run where it was written.
//
// A body is code that runs when the scope is called, and that is as true of a scope written
// inside another as of one written at the top. Its body used to be written straight into the
// outer one and run on the way past — its return firing in the middle of code that had not
// asked for it — so the first of these answered 1 on chain where the program returns 2. It
// compiled, it deployed, and it said nothing.
func TestAScopeWrittenInsideAnotherDoesNotRunOnTheWayPast(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "the outer scope answers its own last expression",
			source: "ident outer = defer { ident inner = defer { 1; }; 2 + feed(0); };",
		},
		{
			// A scope whose last expression is a binding answers the neutral value, which is
			// what the binding is worth here: on a chain a scope is not a value.
			name:   "even when the binding is the last thing in it",
			source: "ident outer = defer { ident inner = defer { 1; }; };",
		},
		{
			name:   "and at any depth",
			source: "ident outer = defer { ident inner = defer { ident deep = defer { 9; }; 1; }; 2 + feed(0); };",
		},
		{
			name:   "the names the outer scope binds are still its own",
			source: "ident outer = defer { ident x = feed(0); ident inner = defer { 1; }; x + x; };",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, tc.source, "outer", []string{"3"}, 0)
		})
	}
}

// A position a scope reads and the call did not fill returns zeros, which is what the
// language answers for reading past what was applied.
//
// On the way in from a transaction that is free: the calldata gives zeros past its end. A
// frame is not free — it is memory an earlier activation already used, so what is left there
// is the last call's values. Two scopes reading two positions, one of them called with one,
// and the second read the first call's second value: the chain answered 501 where the program
// answers 301.
func TestAShortCallReadsZeroOnChainToo(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "a call short by one, after a call that filled the place",
			source: `ident c = defer { feed(0) + feed(1); };
ident b = defer { feed(0) + feed(1); };
ident a = defer { c(100, 200) + b(1); };`,
		},
		{
			name: "short by two",
			source: `ident c = defer { feed(0) + feed(1) + feed(2); };
ident b = defer { feed(0) + feed(1) + feed(2); };
ident a = defer { c(100, 200, 300) + b(1); };`,
		},
		{
			name: "the same scope, called long and then short",
			source: `ident b = defer { feed(0) + feed(1); };
ident a = defer { b(100, 200) + b(1); };`,
		},
		{
			name: "a scope that reads nothing, called with values",
			source: `ident c = defer { feed(0) + feed(1); };
ident b = defer { 7; };
ident a = defer { c(100, 200) + b(1, 2); };`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, tc.source, "a", []string{"0"}, 0)
		})
	}
}

// A scope calls itself, which is what the frame was for.
//
// Nothing was written to make this happen: each activation moves the frame pointer past the
// caller's own memory before writing what it applies, so the call from inside does not see the
// places the call from outside is keeping. What used to stop it was not the missing jump — it
// was that a name had one address for the whole contract, so the second activation wrote over
// the first.
//
// There is no depth written down anywhere, and there should not be. What ends a recursion that
// does not end itself is gas.
func TestAScopeCallsItselfOnChainToo(t *testing.T) {
	const source = `ident down = defer { if feed(0) equals 0 { 0; } else { down(feed(0) - 1) + 1; }; };`

	for _, from := range []string{"0", "1", "3", "9"} {
		t.Run("from "+from, func(t *testing.T) {
			agree(t, source, "down", []string{from}, 0)
		})
	}
}

// Two scopes calling each other, which needs the same thing and one more: a scope that names
// one written after it. The names of every scope are known before any address of one is, so
// which of them was written first says nothing.
func TestTwoScopesCallEachOtherOnChainToo(t *testing.T) {
	const source = `ident even = defer { if feed(0) equals 0 { 1; } else { odd(feed(0) - 1); }; };
ident odd = defer { if feed(0) equals 0 { 0; } else { even(feed(0) - 1); }; };`

	for _, from := range []string{"0", "1", "4", "7"} {
		t.Run("from "+from, func(t *testing.T) {
			agree(t, source, "even", []string{from}, 0)
		})
	}
}

// A program of several files reaches the chain whole: a scope bound in one file is as callable
// by a transaction as one bound in another.
//
// Each file is compiled on its own and numbers its blocks from zero, so joining them moves
// everything the later ones name — and the program runs through all of them in the order their
// dependencies were found, so a scope bound in any of them is reached the same way.
func TestAProgramOfSeveralFilesReachesTheChain(t *testing.T) {
	dir := projectOf(t, map[string]string{
		"src/geometry.ar": "ident area = defer { feed(0) * feed(1); };",
		"src/main.ar":     "use geometry as g;\nident twice = defer { g.area(feed(0), 2); };",
	})

	binary := filepath.Join(dir, "contract.bin")
	if _, err := newSession(t, sessionOpts{}).Build(t.Context(), filepath.Join(dir, "src", "main.ar"), binary); err != nil {
		t.Fatalf("building: %v", err)
	}
	bytecode, err := os.ReadFile(binary)
	if err != nil {
		t.Fatalf("reading the binary: %v", err)
	}

	cfg := &runtime.Config{GasLimit: 10_000_000, Value: big.NewInt(0)}
	_, address, _, err := runtime.Create(bytecode, cfg)
	if err != nil {
		t.Fatalf("deploying: %v", err)
	}

	// The scope bound in the file that was imported, called straight from a transaction. It
	// answers to the name the module gave it, which is how a name crossing a file is written.
	returned, _, err := runtime.Call(address, append(EncodeSelector("geometry.area"), ParseArgs([]string{"6", "7"})...), cfg)
	if err != nil {
		t.Fatalf("calling geometry.area: %v", err)
	}
	if got := decimalOf(returned); got != "42" {
		t.Errorf("geometry.area answered %s, want 42", got)
	}

	// And the scope of the file that imported it, which reaches the other through a call.
	returned, _, err = runtime.Call(address, append(EncodeSelector("twice"), ParseArgs([]string{"21"})...), cfg)
	if err != nil {
		t.Fatalf("calling twice: %v", err)
	}
	if got := decimalOf(returned); got != "42" {
		t.Errorf("twice answered %s, want 42", got)
	}
}

// A block written inside an expression is an expression: control goes into it, it computes a
// value, and control carries on with that value in hand.
//
// It used to end the scope instead. A block opens with the same instruction a scope's body
// does and ends with the same return, so the derivation read the inner return as the outer
// scope's — and everything written after the block was dropped, quietly, from a contract that
// still deployed.
func TestABlockInsideAnExpressionAnswersTheSameOnChainAndOff(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "its value is used after it",
			source: "ident f = defer { ident a = { feed(0) + 1; }; a * 10; };",
		},
		{
			name:   "and the scope returns something else entirely",
			source: "ident f = defer { ident a = { feed(0) + 1; }; feed(0) * 2; };",
		},
		{
			name:   "a block whose value is the scope's answer",
			source: "ident f = defer { { feed(0) + 1; }; };",
		},
		{
			name:   "two of them",
			source: "ident f = defer { ident a = { feed(0) + 1; }; ident b = { a * 2; }; b + a; };",
		},
		{
			name:   "one inside a branch",
			source: "ident f = defer { if feed(0) bigger 0 { ident a = { feed(0) + 1; }; a * 3; } else { 0; }; };",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, tc.source, "f", []string{"4"}, 0)
		})
	}
}

// On a chain a print is the value it was given.
//
// The log has nowhere to go — that is a decision, not a gap — but a print is an expression like
// any other and is worth what it showed, which is what lets one be written into a program that
// already works without changing what that program answers.
//
// It carried on by accident before, and only sometimes: a value another instruction worked out
// was already on the stack, so a print over it happened to leave the right thing there. A value
// the program wrote down was never put on the stack at all, and the contract underflowed the
// first time anything read what the print was worth.
//
// The evaluator prints as it goes and the chain does not, so what is compared here is only what
// the scope returns — the harness runs both and reads the last thing said.
func TestAPrintIsTheValueItWasGivenOnChain(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "a value the program wrote down",
			source: "ident f = defer { ident a = printd 5; a + 10; };",
			want:   "15",
		},
		{
			name:   "a value another instruction worked out",
			source: "ident f = defer { ident a = printb feed(0); a + 10; };",
			want:   "13",
		},
		{
			name:   "read as text, which changes nothing about the value",
			source: "ident f = defer { ident a = printc 26729; a + 1; };",
			want:   "26730",
		},
		{
			name:   "and one whose value nobody reads still returns for the scope",
			source: "ident f = defer { printd feed(0); };",
			want:   "3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decimalOf(onChain(t, tc.source, "f", []string{"3"}, 0)); got != tc.want {
				t.Errorf("the chain answered %s, want %s", got, tc.want)
			}
		})
	}
}

// Every example compiles to bytecode a chain will keep, and deploys.
//
// The examples are run by the evaluator elsewhere, and their declared output is compared there
// too. This is the other half: the same sources through the backend, deployed to an EVM in
// memory. It does not compare what they answer — an example is written to be read, so most of
// them say what they do with print, and print is ignored in compiled code by decision — but a
// contract that will not deploy is a contract that is wrong before anything answers.
//
// It is here rather than beside the other example tests because deploying is what the harness
// does, and because what this guards is the backend rather than the language.
func TestEveryExampleDeploys(t *testing.T) {
	sources, err := exampleSources(t)
	if err != nil {
		t.Fatalf("walking examples: %v", err)
	}
	if len(sources) == 0 {
		t.Fatal("no examples found")
	}

	for _, source := range sources {
		t.Run(filepath.Base(source), func(t *testing.T) {
			// A test file belongs to "aurora test" and holds assertions, which produce no
			// bytecode by decision.
			if strings.HasSuffix(source, ".test.ar") {
				t.Skip("a test file is not built")
			}

			// From where a person would build it: the root of the project it belongs to,
			// since that is what module names resolve from.
			entry := runFrom(t, source)

			binary := filepath.Join(t.TempDir(), "contract.bin")
			if _, err := newSession(t, sessionOpts{}).Build(t.Context(), entry, binary); err != nil {
				t.Fatalf("building: %v", err)
			}
			bytecode, err := os.ReadFile(binary)
			if err != nil {
				t.Fatalf("reading the binary: %v", err)
			}

			cfg := &runtime.Config{GasLimit: 10_000_000, Value: big.NewInt(0)}
			if _, _, _, err := runtime.Create(bytecode, cfg); err != nil {
				t.Fatalf("deploying: %v", err)
			}
		})
	}
}

// A scope can only reach what it bound itself, and reading a name from around it is refused
// rather than written.
//
// A name lives in the frame of the scope that bound it. A scope reading one from outside would
// be reading its own frame at the place that name has in another — and the place a name nobody
// here bound has is the first one, which is either a value applied to this scope or nothing at
// all. So
//
//	ident base = 30;
//	ident show = defer { base * 2; };
//
// answered 0 on a chain where the program returns 60, and said nothing about it. What would
// let it work is a static link, which is a design of its own and written down in
// rfcs/if_and_call.md; until it exists, this is a refusal.
func TestAScopeReadingANameFromAroundItIsRefused(t *testing.T) {
	dir := t.TempDir()
	path := writeAt(t, dir, "contract.ar", "ident base = 30;\nident show = defer { base * 2; };")

	_, err := newSession(t, sessionOpts{}).Build(t.Context(), path, filepath.Join(dir, "c.bin"))
	if err == nil {
		t.Fatal("it wrote a contract that reads a frame nobody filled")
	}
	if !strings.Contains(err.Error(), "bound outside the scope that reads it") {
		t.Errorf("it says %q, and never says the name belongs to another scope", err)
	}
}

// And the same program with the binding moved inside the scope compiles and answers, which is
// what the refusal is telling whoever wrote the other one to do.
func TestAScopeThatBindsWhatItReadsAnswers(t *testing.T) {
	const source = "ident show = defer { ident base = 30; base * 2; };"

	if got := decimalOf(onChain(t, source, "show", []string{}, 0)); got != "60" {
		t.Errorf("the chain answered %s, want 60", got)
	}
}
