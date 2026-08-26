package cli

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/builder/evm"
	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

// The lowering never puts two instructions in the other order unless one of them is Pure.
//
// It is the rule of rfcs/effect.md, run over real programs rather than asserted. The lowering
// holds an instruction back and emits it next to whoever takes its value, which is sound only
// when the value is the whole of what the instruction does — and until now the only thing
// keeping it sound was three lists of opcodes in builder/evm.
//
// The programs below are what the backend carries: a name bound inside a scope, a scope
// calling another, a branch, a shape, the tape operations. They are here rather than in
// builder/evm because getting from source to blocks is wiring, and wiring happens where the
// pipeline is already assembled.
func TestTheLoweringMovesNothingItMayNotMove(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{name: "a name bound inside a scope", source: `ident f = defer { ident x = feed(0); x + feed(1); };`},
		{name: "a name read twice", source: `ident f = defer { ident x = feed(0); x + x; };`},
		{name: "one name reading another", source: `ident f = defer { ident x = feed(0); ident y = x + feed(1); y + x; };`},
		{name: "a binding as the last expression", source: `ident f = defer { ident x = feed(0); ident y = x + 1; };`},
		{name: "a value written down on the left of a name", source: `ident f = defer { ident x = feed(0); 10 - x; };`},
		{name: "a subtraction of two names", source: `ident f = defer { ident a = feed(0); ident b = feed(1); a - b; };`},
		{name: "a branch over names", source: `ident f = defer { ident x = feed(0); if x bigger 2 { x - 1; } else { x + 1; }; };`},
		{name: "a scope calling another", source: `ident g = defer { feed(0) * 2; };` + "\n" + `ident f = defer { ident x = feed(0); g(x) + x; };`},
		{name: "a shape and a field", source: `shape P { a, b };` + "\n" + `ident f = defer { ident p = P{feed(0), feed(1)}; p.a + p.b; };`},
		{name: "the tape operations", source: "ident f = defer { ident x = feed(0); ident a = head x 1; ident b = tail x 1; a + b; };"},
		{name: "a print over a name", source: `ident f = defer { ident x = feed(0); printd x; };`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeAt(t, dir, "contract.ar", tc.source)
			program, err := newSession(t, sessionOpts{}).compile(path)
			if err != nil {
				t.Fatalf("compiling: %v", err)
			}

			held := 0
			for _, block := range program.Blocks {
				lowered := evm.LowerBlock(block, 8).Insts
				if moved := crossedWrongly(block.Insts, lowered); moved != "" {
					t.Errorf("block %d: %s", block.ID, moved)
				}
				held += notPure(lowered)
			}

			// Or the case proves nothing: a program of only Pure instructions passes the
			// rule by having nothing to ask it about.
			if held < 2 {
				t.Errorf("only %d instructions here are not Pure, so the rule was never asked", held)
			}
		})
	}
}

// crossedWrongly names a pair the lowering put in the other order that the rule forbids, and
// nothing when it kept every one of them.
//
// Only the instructions that are not Pure are compared, and by where they sat in the block
// they came from: what the rule is about is which of two ran first, and a Pure instruction
// moving past anything is exactly what the lowering exists to do.
func crossedWrongly(before, after []ir.Instruction) string {
	// By label and opcode, because an instruction holds slices and cannot be a map key. A
	// label is unique in its block, and the opcode is there because the lowering writes a
	// save under the label of the binding it stands in for.
	was := make(map[string]int, len(before))
	for at, inst := range before {
		was[keyOf(inst)] = at
	}

	held := make([]ir.Instruction, 0, len(after))
	for _, inst := range after {
		if ir.EffectOf(inst.GetOpCode()) == ir.Pure {
			continue
		}
		if _, written := was[keyOf(inst)]; !written {
			// The lowering writes instructions of its own, which were in no order to begin
			// with and so cannot have left one.
			continue
		}
		held = append(held, inst)
	}

	// Every pair, not only the neighbours: an instruction can be moved past more than one, and
	// the rule is about which two ended up in the other order however far apart they were.
	for i := 0; i < len(held); i++ {
		for j := i + 1; j < len(held); j++ {
			first, second := held[i], held[j]
			if was[keyOf(first)] <= was[keyOf(second)] {
				continue
			}
			if ir.MayCross(ir.EffectOf(first.GetOpCode()), ir.EffectOf(second.GetOpCode())) {
				continue
			}
			return describeCrossing(second, first, was)
		}
	}
	return ""
}

// notPure counts the instructions in a block the rule has anything to say about.
func notPure(insts []ir.Instruction) int {
	count := 0
	for _, inst := range insts {
		if ir.EffectOf(inst.GetOpCode()) != ir.Pure {
			count++
		}
	}
	return count
}

func keyOf(inst ir.Instruction) string {
	return byteutil.ToHex(inst.GetLabel()) + ":" + itoa(int(inst.GetOpCode()))
}

// describeCrossing says which two instructions swapped, and what each of them does, because
// the opcode numbers on their own would send whoever reads the failure to a table.
func describeCrossing(first, second ir.Instruction, was map[string]int) string {
	return strings.Join([]string{
		"the lowering swapped two instructions the rule holds:",
		describeEffect(second), "was written at", itoa(was[keyOf(second)]),
		"and now runs before", describeEffect(first), "written at", itoa(was[keyOf(first)]),
	}, " ")
}

func describeEffect(inst ir.Instruction) string {
	names := map[ir.Effect]string{ir.Reads: "a read", ir.Writes: "a write", ir.Escapes: "an escape"}
	return names[ir.EffectOf(inst.GetOpCode())] + " (opcode " + itoa(int(inst.GetOpCode())) + ")"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 4)
	for ; n > 0; n /= 10 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
	}
	return string(digits)
}
