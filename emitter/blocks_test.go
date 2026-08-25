package emitter

import (
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

// listOf answers the instruction list the emitter builds on its way to the blocks. It is not
// something a caller is given — the blocks are — and it exists here because what this file
// checks is that the crossing between the two loses nothing.
func listOf(t *testing.T, source string) []ir.Instruction {
	t.Helper()

	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	tree, err := parser.New().Parse(parser.ParseInput{Filename: "main.ar", Tokens: tokens})
	if err != nil {
		t.Fatalf("parser: %v", err)
	}

	tc := 0
	insts := make([]ir.Instruction, 0)
	for _, node := range tree.Nodes {
		EmitInstruction(&tc, &insts, node, 0)
	}
	return insts
}

// structure names the opcodes that do not survive the crossing: each of them said where
// control goes or how far something reached, which is what a block and a terminator say now.
var structure = map[byte]bool{
	ir.OpDefer:      true,
	ir.OpBeginScope: true,
	ir.OpIf:         true,
	ir.OpJump:       true,
	ir.OpReturn:     true,
}

// A scope is the one of them that leaves something behind: it said how long its body was, and
// that is gone, but a scope is a value like any other and what it is worth crosses over under
// the same name.
var leavesAValue = map[byte]bool{ir.OpDefer: true}

// The blocks describe the same program: every instruction that computes a value is in exactly
// one of them, and nothing else is.
//
// This is what makes the block form trustworthy while both forms exist. A derivation that
// quietly dropped an instruction, or kept one twice, would still produce blocks that look
// well-formed — and the consumer that noticed would be a contract answering the wrong thing.
func TestBlocksHoldEveryInstructionThatComputesAValue(t *testing.T) {
	const source = `shape Point { x, y };
ident p = Point{10, 20};
ident area = defer { ident q = feed(0) as Point; q.x * q.y; };
ident t = [1, 2, 3];
printb head t 2;
ident answer = defer { if feed(0) bigger 0 { area(p) + 1; } else { 0; }; };
printd answer(1);`

	insts := listOf(t, source)

	wanted := make(map[string]int)
	for _, inst := range insts {
		if !structure[inst.GetOpCode()] || leavesAValue[inst.GetOpCode()] {
			wanted[byteutil.ToHex(inst.GetLabel())]++
		}
	}

	found := make(map[string]int)
	for _, block := range ir.BlocksOf(insts) {
		for _, inst := range block.Insts {
			if structure[inst.GetOpCode()] {
				t.Errorf("block %d holds %s, which is structure", block.ID, ir.ResolveOpCode(inst.GetOpCode()))
			}
			found[byteutil.ToHex(inst.GetLabel())]++
		}
	}

	for label, times := range wanted {
		if found[label] != times {
			t.Errorf("the label %s is in the list %d times and in the blocks %d", label, times, found[label])
		}
	}
	for label := range found {
		if wanted[label] == 0 {
			t.Errorf("the blocks hold %s, which the list does not", label)
		}
	}
}

// Every block a terminator names exists, and each kind of ending names as many blocks as it
// chooses between: a return names none, going names one, choosing names two.
func TestEveryTerminatorNamesBlocksThatExist(t *testing.T) {
	const source = `ident sign = defer {
  if feed(0) bigger 0 { 1; } else { if feed(0) smaller 0 { 2; } else { 0; }; };
};
printd sign(5);`

	blocks := compile(t, source).Blocks

	wanted := map[ir.TermKind]int{ir.Ret: 0, ir.Br: 1, ir.BrIf: 2}
	for _, block := range blocks {
		term := block.Term
		if got := len(term.Targets); got != wanted[term.Kind] {
			t.Errorf("block %d ends naming %d blocks, want %d", block.ID, got, wanted[term.Kind])
		}
		for _, target := range term.Targets {
			if int(target.Block) < 0 || int(target.Block) >= len(blocks) {
				t.Errorf("block %d goes to block %d, and there are %d", block.ID, target.Block, len(blocks))
			}
		}
	}
}

// A branch becomes four blocks — the run before it, an arm each, and the one they meet at —
// and both arms hand their value to the same block. That is what makes an "if" an expression,
// said as structure rather than as a place two consumers agree on.
func TestBothArmsHandTheirValueToTheSameBlock(t *testing.T) {
	const source = `ident answer = defer { if feed(0) bigger 0 { 10; } else { 20; }; };`

	blocks := compile(t, source).Blocks

	var chose *ir.Block
	for at := range blocks {
		if blocks[at].Term.Kind == ir.BrIf {
			chose = &blocks[at]
		}
	}
	if chose == nil {
		t.Fatal("no block chooses, and the program has an if")
	}

	arms := chose.Term.Targets
	meetings := make([]ir.BlockID, 0, 2)
	for _, arm := range arms {
		term := blocks[arm.Block].Term
		if term.Kind != ir.Br {
			t.Fatalf("an arm ends with %v, want it going to where they meet", term.Kind)
		}
		if len(term.Targets[0].Args) != 1 {
			t.Errorf("an arm hands over %d values, want the one it computed", len(term.Targets[0].Args))
		}
		meetings = append(meetings, term.Targets[0].Block)
	}

	if meetings[0] != meetings[1] {
		t.Errorf("the arms meet at %d and %d, want one block", meetings[0], meetings[1])
	}
	meet := blocks[meetings[0]]
	if len(meet.Params) != 1 {
		t.Errorf("the block they meet at takes %d values, want the one each arm hands it", len(meet.Params))
	}
	// It names the one it takes, and that name is what the value is known as after the branch:
	// here the scope answers with it, without knowing which arm computed it.
	if len(meet.Params) == 1 && string(meet.Term.Value.Bytes()) != string(meet.Params[0]) {
		t.Errorf("it answers with %q and takes %q, want them to be the same value",
			meet.Term.Value.Bytes(), meet.Params[0])
	}
}

// A scope is a block, and how many values it takes is what its body reads: the highest
// position it feeds, plus one. A scope written inside it reads its own vector, so its feeds
// say nothing about this one.
func TestAScopeBlockTakesWhatItsBodyReads(t *testing.T) {
	const source = `ident two = defer { feed(0) + feed(1); };
ident none = defer { 7; };
ident outer = defer { ident inner = defer { feed(0) + feed(1) + feed(2); }; feed(0); };`

	blocks := compile(t, source).Blocks

	found := make(map[int]int)
	for _, block := range blocks {
		found[len(block.Params)]++
	}

	// two takes 2, none takes 0, inner takes 3, outer takes 1 — and the top of the program
	// takes nothing.
	for _, params := range []int{2, 3, 1} {
		if found[params] == 0 {
			t.Errorf("no block takes %d values, and a scope in the source reads that many", params)
		}
	}
}
