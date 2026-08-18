package cli

import (
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

// offChain answers what the evaluator makes of the same call.
//
// The call has to be written into the source, because there is no way to ask the evaluator
// for one scope by name with these arguments — the closest thing, "aurora call", only speaks
// to a network. That is the gap this harness makes visible.
func offChain(t *testing.T, source, call string, tapeSize int) string {
	t.Helper()

	path := writeAt(t, t.TempDir(), "program.ar", source+"\nprintd "+call+";\n")
	out := &strings.Builder{}
	if err := newSession(t, sessionOpts{tapeSize: tapeSize, stdout: out}).Run(t.Context(), path); err != nil {
		t.Fatalf("running: %v", err)
	}
	return strings.TrimSpace(out.String())
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
		call string
	}{
		{name: "small", args: []string{"1", "2"}, call: "add(1, 2)"},
		{name: "larger", args: []string{"1000", "337"}, call: "add(1000, 337)"},
		{name: "zero", args: []string{"0", "0"}, call: "add(0, 0)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			returned := onChain(t, source, "add", tc.args, 0)
			want := offChain(t, source, tc.call, 0)

			if got := decimalOf(returned); got != want {
				t.Errorf("the chain answered %s and the evaluator %s", got, want)
			}
		})
	}
}

// A contract holds as many scopes as it has names, and the selector is what tells them apart.
func TestEachScopeAnswersForItself(t *testing.T) {
	const source = `ident add = defer { feed(0) + feed(1); };
ident multiply = defer { feed(0) * feed(1); };`

	cases := []struct {
		function string
		call     string
	}{
		{function: "add", call: "add(6, 7)"},
		{function: "multiply", call: "multiply(6, 7)"},
	}

	for _, tc := range cases {
		t.Run(tc.function, func(t *testing.T) {
			returned := onChain(t, source, tc.function, []string{"6", "7"}, 0)
			want := offChain(t, source, tc.call, 0)

			if got := decimalOf(returned); got != want {
				t.Errorf("the chain answered %s and the evaluator %s", got, want)
			}
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
