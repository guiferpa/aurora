package cli

import (
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

// A name the contract does not have reaches the no-match STOP, which answers with nothing —
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
			// 0 — but 256 / 2 is 128, and cutting that afterwards still answers 128.
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

// A name bound inside a scope, which is what a body of more than one line is made of.
//
// It compiled before this and answered nothing: the lowering handed values to arithmetic and
// to a return and to nothing else, so the one the binding meant to store was never pushed and
// its MSTORE ran on an empty stack. The contract deployed, the call succeeded, and the answer
// came back empty — which reads as zero, and reads as nothing being wrong.
func TestALocalInsideAScopeAnswersTheSameOnChainAndOff(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		args   []string
	}{
		{
			name:   "read twice",
			source: "ident area = defer {\n  ident side = feed(0);\n  side * side;\n};",
			args:   []string{"5"},
		},
		{
			name:   "two of them, and one reads the other",
			source: "ident total = defer {\n  ident base = feed(0) + feed(1);\n  ident twice = base * 2;\n  twice + base;\n};",
			args:   []string{"3", "4"},
		},
		{
			name:   "one that nothing reads",
			source: "ident area = defer {\n  ident unused = feed(1);\n  feed(0) * feed(0);\n};",
			args:   []string{"6", "9"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agree(t, tc.source, functionOf(tc.source), tc.args, 0)
		})
	}
}

// functionOf reads the name of the first scope a source binds, which is the one these call.
func functionOf(source string) string {
	name := strings.TrimPrefix(source, "ident ")
	return strings.TrimSpace(name[:strings.Index(name, "=")])
}
