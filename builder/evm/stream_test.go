package evm

import (
	"testing"
)

// Bytecode is a stream, not a list: an opcode that takes an operand is followed by that
// many bytes, and everything after them is read as the next opcode. Get the width wrong by
// one and nothing complains — the following bytes are still valid opcodes, just the wrong
// ones, and the contract does something else entirely.
//
// This is the one thing reading the bytes back can prove and writing them cannot: the
// writer's own account says what it meant, and this says what came out. It used to be
// proved by eye, through a disassembler behind a flag someone had to remember to pass;
// here it is proved on every run.

// pushWidth returns how many operand bytes an opcode carries, and whether it carries any.
// The push opcodes are contiguous from PUSH1 to PUSH32, which is what makes this arithmetic
// rather than a table.
func pushWidth(op byte) (int, bool) {
	if op < OpPush1 || op > OpPush32 {
		return 0, false
	}
	return int(op-OpPush1) + 1, true
}

// walk reads the stream to its end, answering where it stopped: the number of opcodes read,
// and the operand bytes still owed when the stream ran out.
func walk(bytecode []byte) (opcodes, owed int) {
	for i := 0; i < len(bytecode); {
		opcodes++
		width, carries := pushWidth(bytecode[i])
		i++
		if !carries {
			continue
		}
		if i+width > len(bytecode) {
			return opcodes, i + width - len(bytecode)
		}
		i += width
	}
	return opcodes, 0
}

func TestBytecodeIsAWellFormedStream(t *testing.T) {
	sources := []string{
		"ident a = 1;\n",
		"ident add = defer { feed(0) + feed(1); };\n",
		"ident add = defer { feed(0) + feed(1); };\nident mul = defer { feed(0) * feed(1); };\n",
	}

	for _, tapeSize := range []int{1, 8, 32} {
		for _, source := range sources {
			bytecode := build(t, source, tapeSize)

			opcodes, owed := walk(bytecode)
			if owed != 0 {
				t.Errorf("%d-byte tapes, %q: the stream ends owing %d operand bytes after %d opcodes",
					tapeSize, source, owed, opcodes)
			}
			if opcodes == 0 {
				t.Errorf("%d-byte tapes, %q: no opcode at all", tapeSize, source)
			}
		}
	}
}

// The walk has to be able to fail, or it proves nothing: a push that claims more bytes
// than the stream has is exactly the desynchronisation being guarded against.
func TestTheWalkNoticesATruncatedPush(t *testing.T) {
	truncated := []byte{OpPush4, 0x01, 0x02}

	if _, owed := walk(truncated); owed != 2 {
		t.Errorf("a PUSH4 with two bytes should end owing 2, got %d", owed)
	}
}
