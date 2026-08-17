package evm

import (
	"bytes"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

// build compiles source all the way to EVM bytecode, which is what actually gets deployed.
func build(t *testing.T, source string, tapeSize int) []byte {
	t.Helper()

	tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	tree, err := parser.New(parser.NewParserOptions{Filename: "main.ar", Tokens: tokens, TapeSize: tapeSize}).Parse()
	if err != nil {
		t.Fatalf("parser: %v", err)
	}
	insts, err := emitter.New(emitter.NewEmitterOptions{TapeSize: tapeSize}).Emit(tree)
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}

	bytecode, err := NewBuilder(insts, NewBuilderOptions{TapeSize: tapeSize}).Build()
	if err != nil {
		t.Fatalf("builder: %v", err)
	}
	return bytecode
}

// Every contract opens with the instantiate block, which copies the runtime code out and
// returns it — that is what the chain stores.
func TestBuildStartsWithTheInstantiateBlock(t *testing.T) {
	code := build(t, "ident a = 1;\n", byteutil.DefaultTapeSize)

	if len(code) <= INSTANTIATE_BLOCK_SIZE {
		t.Fatalf("bytecode is %d bytes, too short to hold the instantiate block", len(code))
	}
	// PUSH1 <size> PUSH1 0x0c PUSH1 0x00 CODECOPY PUSH1 <size> PUSH1 0x00 RETURN
	want := []byte{OpPush1, byte(len(code) - INSTANTIATE_BLOCK_SIZE), OpPush1, 0x0c, OpPush1, 0x00, OpCodeCopy}
	if !bytes.Equal(code[:len(want)], want) {
		t.Errorf("instantiate block = %s, want %s",
			byteutil.ToUpperHex(code[:len(want)]), byteutil.ToUpperHex(want))
	}
	if code[INSTANTIATE_BLOCK_SIZE-1] != OpReturn {
		t.Errorf("the instantiate block should end in RETURN, got %#x", code[INSTANTIATE_BLOCK_SIZE-1])
	}
}

// The size announced by the instantiate block has to match what follows it, or the chain
// stores the wrong slice of code.
func TestBuildAnnouncesTheRuntimeSize(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{name: "root code only", source: "ident a = 1 + 2;\n"},
		{name: "one callable", source: "ident add = defer { feed(0) + feed(1); };\n"},
		{name: "callable and root code", source: "ident add = defer { feed(0) + 1; };\nident b = 2;\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code := build(t, tc.source, byteutil.DefaultTapeSize)
			announced := int(code[1])
			if got := len(code) - INSTANTIATE_BLOCK_SIZE; announced != got {
				t.Errorf("the block announces %d runtime bytes, %d follow", announced, got)
			}
		})
	}
}

// A defer bound to a name becomes a public entry point: the dispatcher compares the first
// four bytes of calldata against the keccak of the name and jumps to its code.
func TestBuildEmitsADispatcherPerCallable(t *testing.T) {
	code := build(t, "ident add = defer { feed(0) + feed(1); };\n", byteutil.DefaultTapeSize)
	runtime := code[INSTANTIATE_BLOCK_SIZE:]

	if runtime[0] != OpPush1 || runtime[1] != 0x00 || runtime[2] != OpCallDataLoad {
		t.Errorf("a dispatcher should start by loading calldata, got %s", byteutil.ToUpperHex(runtime[:3]))
	}
	if runtime[DISPATCHER_BYTES_SIZE-1] != OpJumpIf {
		t.Errorf("a dispatcher should end in JUMPI, got %#x", runtime[DISPATCHER_BYTES_SIZE-1])
	}
	// A call that matches nothing stops rather than falling into the first body.
	if runtime[DISPATCHER_BYTES_SIZE] != OpStop {
		t.Errorf("expected STOP after the dispatchers, got %#x", runtime[DISPATCHER_BYTES_SIZE])
	}
	// The body it jumps to is a JUMPDEST.
	if runtime[DISPATCHER_BYTES_SIZE+NO_MATCH_DISPATCHER_SIZE] != OpJumpDestiny {
		t.Error("the code a dispatcher jumps to must start with JUMPDEST")
	}
}

func TestBuildEmitsOneDispatcherPerName(t *testing.T) {
	code := build(t, `ident add = defer { feed(0) + feed(1); };
ident sub = defer { feed(0) - feed(1); };
`, byteutil.DefaultTapeSize)
	runtime := code[INSTANTIATE_BLOCK_SIZE:]

	for i := 0; i < 2; i++ {
		at := i * DISPATCHER_BYTES_SIZE
		if runtime[at] != OpPush1 || runtime[at+2] != OpCallDataLoad {
			t.Errorf("dispatcher %d is malformed: %s", i, byteutil.ToUpperHex(runtime[at:at+DISPATCHER_BYTES_SIZE]))
		}
	}
	if runtime[2*DISPATCHER_BYTES_SIZE] != OpStop {
		t.Error("expected the no-match STOP after both dispatchers")
	}
}

// Without a callable there is nothing to dispatch on, so the runtime is the root code.
func TestBuildWithoutCallables(t *testing.T) {
	code := build(t, "ident a = 1 + 2;\n", byteutil.DefaultTapeSize)
	runtime := code[INSTANTIATE_BLOCK_SIZE:]

	if runtime[0] == OpPush1 && len(runtime) > 2 && runtime[2] == OpCallDataLoad {
		t.Error("no dispatcher should be emitted when there is no callable")
	}
	if runtime[len(runtime)-1] != OpStop {
		t.Errorf("runtime should end in STOP, got %#x", runtime[len(runtime)-1])
	}
}

// The push opcode carries the tape width, so the same source compiles to different
// bytecode depending on it.
func TestBuildFollowsTheTapeSize(t *testing.T) {
	cases := []struct {
		tapeSize int
		opcode   byte
	}{
		{tapeSize: 1, opcode: OpPush1},
		{tapeSize: 2, opcode: OpPush2},
		{tapeSize: 8, opcode: OpPush8},
		{tapeSize: 32, opcode: OpPush32},
	}

	for _, tc := range cases {
		t.Run(byteutil.ToHex([]byte{tc.opcode}), func(t *testing.T) {
			code := build(t, "ident add = defer { feed(0) + 1; };\n", tc.tapeSize)
			if !bytes.Contains(code, []byte{tc.opcode}) {
				t.Errorf("expected a PUSH of width %d (%#x) in %s",
					tc.tapeSize, tc.opcode, byteutil.ToUpperHex(code))
			}
		})
	}
}

func TestGetRuntimeCodeLength(t *testing.T) {
	cases := []struct {
		name string
		code *RuntimeCode
		want int
	}{
		{name: "empty", code: &RuntimeCode{}, want: 0},
		{
			name: "root only",
			code: &RuntimeCode{Root: bytes.NewBuffer([]byte{1, 2, 3})},
			want: 3,
		},
		{
			name: "one dispatcher",
			code: &RuntimeCode{Dispatchers: []Dispatcher{{Code: bytes.NewBuffer([]byte{1, 2})}}},
			want: DISPATCHER_BYTES_SIZE + NO_MATCH_DISPATCHER_SIZE + 2,
		},
		{
			name: "dispatchers and root",
			code: &RuntimeCode{
				Dispatchers: []Dispatcher{
					{Code: bytes.NewBuffer([]byte{1, 2})},
					{Code: bytes.NewBuffer([]byte{3})},
				},
				Root: bytes.NewBuffer([]byte{4, 5, 6, 7}),
			},
			want: 2*DISPATCHER_BYTES_SIZE + NO_MATCH_DISPATCHER_SIZE + 3 + 4,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GetRuntimeCodeLength(tc.code); got != tc.want {
				t.Errorf("GetRuntimeCodeLength = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestToOpByte(t *testing.T) {
	cases := []struct {
		in   uint32
		want []byte
	}{
		{in: 0x60, want: []byte{0x60}},
		{in: 0x6001, want: []byte{0x60, 0x01}},
		{in: 0, want: []byte{0, 0, 0, 0}},
	}
	for _, tc := range cases {
		t.Run(byteutil.ToHex(tc.want), func(t *testing.T) {
			if got := ToOpByte(tc.in); !bytes.Equal(got, tc.want) {
				t.Errorf("ToOpByte(%#x) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// The disassembler names opcodes for the builder log, so a wrong name sends whoever is
// debugging bytecode down the wrong path.
func TestResolveOpCode(t *testing.T) {
	cases := []struct {
		op   byte
		want string
	}{
		{op: OpStop, want: "STOP"},
		{op: OpAdd, want: "ADD"},
		{op: OpMul, want: "MUL"},
		{op: OpSub, want: "SUB"},
		{op: OpDiv, want: "DIV"},
		{op: OpPush1, want: "PUSH1"},
		{op: OpPush8, want: "PUSH8"},
		{op: OpPush32, want: "PUSH32"},
		{op: OpJumpIf, want: "JUMPI"},
		{op: OpJumpDestiny, want: "JUMPDEST"},
		{op: OpReturn, want: "RETURN"},
		{op: OpCallDataLoad, want: "CALLDATALOAD"},
		{op: OpMemoryStore, want: "MSTORE"},
		{op: OpMemoryLoad, want: "MLOAD"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := ResolveOpCode(tc.op); got != tc.want {
				t.Errorf("ResolveOpCode(%#x) = %q, want %q", tc.op, got, tc.want)
			}
		})
	}
}

func TestResolveOpCodeOfSomethingUnknown(t *testing.T) {
	if got := ResolveOpCode(0xFE); got == "" {
		t.Error("an unknown opcode should still get a name")
	}
}

func TestWriteNoMatchDispatcher(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	if _, err := WriteNoMatchDispatcher(buf); err != nil {
		t.Fatalf("WriteNoMatchDispatcher: %v", err)
	}
	if got := buf.Bytes(); len(got) != 1 || got[0] != OpStop {
		t.Errorf("got %v, want a single STOP", got)
	}
}

// Nothing to dispatch on means no dispatcher block and no no-match stop.
func TestWriteDispatchersWithNone(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	written, err := WriteDispatchers(buf, nil)
	if err != nil {
		t.Fatalf("WriteDispatchers: %v", err)
	}
	if written != 0 || buf.Len() != 0 {
		t.Errorf("wrote %d bytes, want none", buf.Len())
	}
}

func TestWriteDispatchersReportsItsSize(t *testing.T) {
	dispatchers := []Dispatcher{
		{Selector: []byte("add"), Code: bytes.NewBuffer([]byte{1}), Length: 1},
		{Selector: []byte("sub"), Code: bytes.NewBuffer([]byte{2}), Offset: 2, Length: 1},
	}

	buf := bytes.NewBuffer(nil)
	written, err := WriteDispatchers(buf, dispatchers)
	if err != nil {
		t.Fatalf("WriteDispatchers: %v", err)
	}

	want := 2*DISPATCHER_BYTES_SIZE + NO_MATCH_DISPATCHER_SIZE
	if written != want {
		t.Errorf("reported %d bytes, want %d", written, want)
	}
	if buf.Len() != want {
		t.Errorf("wrote %d bytes, want %d", buf.Len(), want)
	}
}

// Two names must not share a selector, or a call reaches the wrong code.
func TestDispatcherSelectorsDiffer(t *testing.T) {
	code := build(t, `ident add = defer { feed(0); };
ident sub = defer { feed(0); };
`, byteutil.DefaultTapeSize)
	runtime := code[INSTANTIATE_BLOCK_SIZE:]

	// The selector sits after PUSH1 00 CALLDATALOAD PUSH1 e0 SHR PUSH4.
	const selectorAt = 8
	first := runtime[selectorAt : selectorAt+4]
	second := runtime[DISPATCHER_BYTES_SIZE+selectorAt : DISPATCHER_BYTES_SIZE+selectorAt+4]

	if bytes.Equal(first, second) {
		t.Errorf("both dispatchers use the selector %s", byteutil.ToUpperHex(first))
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	const source = "ident add = defer { feed(0) + feed(1); };\nident a = 1;\n"

	first := build(t, source, byteutil.DefaultTapeSize)
	second := build(t, source, byteutil.DefaultTapeSize)

	if !bytes.Equal(first, second) {
		t.Error("the same source produced different bytecode")
	}
}

func TestWriteCodeEndsInStop(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	if _, err := WriteCode(buf, NewIdentManager(), nil, byteutil.DefaultTapeSize); err != nil {
		t.Fatalf("WriteCode: %v", err)
	}
	got := buf.Bytes()
	if len(got) != 1 || got[0] != OpStop {
		t.Errorf("got %v, want a single STOP", got)
	}
}

func TestWriteBodyCodeWithoutRoot(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	dispatchers := []Dispatcher{{Code: bytes.NewBuffer([]byte{1, 2, 3})}}

	if _, err := WriteBodyCode(buf, dispatchers, nil); err != nil {
		t.Fatalf("WriteBodyCode: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), []byte{1, 2, 3}) {
		t.Errorf("got %v, want the dispatcher body", buf.Bytes())
	}
}

// The compiled name reaches the bytecode as a selector, so a change in how names are
// hashed shows up here.
func TestSelectorComesFromTheName(t *testing.T) {
	code := build(t, "ident transfer = defer { feed(0); };\n", byteutil.DefaultTapeSize)

	if !strings.Contains(byteutil.ToUpperHex(code), "63") { // PUSH4 introduces the selector
		t.Error("expected a PUSH4 carrying the selector")
	}
}

// Negating is subtracting from zero, and on chain it has to be exactly that: the bytecode
// for -x is the bytecode for 0 - x, opcode for opcode. The emitter desugars it precisely so
// this backend needs nothing new — if the two ever diverge, something started treating the
// unary form as its own operation without teaching the writer about it.
func TestNegationBuildsAsSubtractionFromZero(t *testing.T) {
	for _, tapeSize := range []int{1, 8, 32} {
		negated := build(t, "ident neg = defer { -feed(0); };\n", tapeSize)
		subtracted := build(t, "ident neg = defer { 0 - feed(0); };\n", tapeSize)

		if !bytes.Equal(negated, subtracted) {
			t.Errorf("%d-byte tapes:\n-feed(0)    = %x\n0 - feed(0) = %x", tapeSize, negated, subtracted)
		}
	}
}
