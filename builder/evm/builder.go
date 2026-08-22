// Package evm compiles the emitter's IR to EVM bytecode.
//
// Architecture: the Builder is intended to be mechanical (emit opcodes, resolve offsets).
// Stack order and LIFO semantics should be guaranteed by a separate Lowering phase that
// consumes IR and produces a stream already in EVM stack order. See
// docs/compiler_pipeline_and_lowering.md for the target pipeline (IR → Lowering → Builder).
package evm

import (
	"bytes"
	"fmt"
	"io"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

const (
	BYTE_SIZE = 8
	// DISPATCHER_BYTES_SIZE is what one entry of the dispatcher measures, added up from its
	// parts rather than written as a number: the address of every body is counted from it, so
	// a push changing size and this staying behind would put every jump one byte off per
	// entry — and it would do that quietly.
	DISPATCHER_BYTES_SIZE    = PUSH_ONE_SIZE + 1 + PUSH_ONE_SIZE + 1 + PUSH_FOUR_SIZE + 1 + PUSH_TWO_SIZE + 1
	NO_MATCH_DISPATCHER_SIZE = 1
	CALLDATA_SLOT_READABLE   = 32
	MEMORY_SLOT_SIZE         = 32
	// PUSH_ONE_SIZE and PUSH_TWO_SIZE are the opcode and what it carries.
	PUSH_ONE_SIZE = 2
	PUSH_TWO_SIZE = 3
	// PUSH_FOUR_SIZE is the opcode and the four bytes of a selector.
	PUSH_FOUR_SIZE = 5
	// INSTANTIATE_BLOCK_SIZE is what the constructor measures, and the runtime begins right
	// after it — so the block carries this number inside itself, as the offset it copies
	// from. It is added up here rather than written as a literal, because it was the same
	// number in two places and a push changing size meant remembering both.
	INSTANTIATE_BLOCK_SIZE = PUSH_TWO_SIZE + PUSH_ONE_SIZE + PUSH_ONE_SIZE + 1 + PUSH_TWO_SIZE + PUSH_ONE_SIZE + 1
	// MAX_CONTRACT_SIZE is what a chain will keep: 24,576 bytes, by EIP-170. A runtime past
	// it is refused rather than written, because writing it produces a binary that deploys
	// and is not the program.
	MAX_CONTRACT_SIZE = 24576
	// FRAME_POINTER is the slot of memory holding where the running scope's frame starts.
	// Everything a scope keeps — the values applied to it, and the names it binds — lives at
	// an offset from there, so two activations of one scope do not share a place.
	FRAME_POINTER = 0x00
	// RETURN_SCRATCH is where a value is put to be handed to the chain. It is not part of a
	// frame: it is written and returned in the same breath, and nothing reads it after.
	RETURN_SCRATCH = 0x20
	// FRAME_BASE is where the first frame starts, past the two slots above.
	FRAME_BASE = 0x40
	// FRAME_START_SIZE is what writing where the first frame begins measures: a push of the
	// address, a push of the slot, and the store.
	FRAME_START_SIZE = PUSH_TWO_SIZE + PUSH_ONE_SIZE + 1
	// ANSWER_SIZE is what handing a value to the chain measures.
	ANSWER_SIZE = PUSH_ONE_SIZE + 1 + PUSH_ONE_SIZE + PUSH_ONE_SIZE + 1
)

type Dispatcher struct {
	Selector []byte
	Offset   int
	Length   int
	Code     *bytes.Buffer
	// Body is what the code was written from, kept so it can be written again once where it
	// lands is known.
	Body []ir.Instruction
}

type RuntimeCode struct {
	Root        *bytes.Buffer
	Dispatchers []Dispatcher
}

type Builder struct {
	tapeSize int
	cursor   int
	insts    []ir.Instruction
	operands [][]byte
}

func (b *Builder) GetInstruction() ir.Instruction {
	return b.insts[b.cursor]
}

// PickDeferAtCursor tries to parse a deferred scope at the given cursor.
// If insts[cursor] is OpDefer with a valid body and the next instruction is OpIdent,
// it returns the Dispatcher (with Offset and Length set), the cursor position after
// the defer body (pointing at the OpIdent), and true. Otherwise returns (nil, cursor, false).
// Does not mutate b.cursor.
func (b *Builder) PickDeferAtCursor(cursor int, offset int) (d *Dispatcher, nextCursor int, ok bool) {
	if cursor >= len(b.insts) {
		return nil, cursor, false
	}
	inst := b.insts[cursor]
	if inst.GetOpCode() != ir.OpDefer {
		return nil, cursor, false
	}

	// OpDefer layout: [OpDefer] [body of length N] [OpIdent]. Right operand = N (body length in instructions).
	bodylength := byteutil.ToUint64(inst.GetRight().Bytes())
	end := cursor + 1 + int(bodylength)
	if end > len(b.insts) {
		return nil, cursor, false
	}
	body := b.insts[cursor+1 : end]
	body = Lowering(body, b.tapeSize)

	// Written once to find out how long it is, and once more when where it lands is known —
	// a jump inside it carries an address in the contract, and that address depends on how
	// many scopes come before it, which is not known until they have all been found.
	code := bytes.NewBuffer(make([]byte, 0))
	if err := WriteScope(code, body, b.tapeSize, 0); err != nil {
		return nil, cursor, false
	}

	// Defer must be assigned to an ident (e.g. "ident f = defer { ... }"); that OpIdent is the selector.
	if end >= len(b.insts) {
		return nil, cursor, false
	}
	selectorInst := b.insts[end]
	if selectorInst.GetOpCode() != ir.OpIdent {
		return nil, cursor, false
	}
	selector := selectorInst.GetLeft().Bytes()

	// Prepend OpJumpDestiny so the EVM can jump to this block when the selector matches.
	d = &Dispatcher{
		Selector: selector,
		Code:     code,
		Offset:   offset,
		Length:   code.Len(),
		Body:     body,
	}
	return d, end, true
}

func (b *Builder) PickRuntimeCode() (*RuntimeCode, error) {
	dispatchers := make([]Dispatcher, 0)
	rootinsts := make([]ir.Instruction, 0)
	offset := 0

	for b.cursor < len(b.insts) {
		inst := b.GetInstruction()
		if d, nextCursor, ok := b.PickDeferAtCursor(b.cursor, offset); ok {
			dispatchers = append(dispatchers, *d)
			offset += d.Length
			// Skip the OpIdent that assigns the defer to a variable; it has no EVM meaning (selector is already in the dispatcher).
			b.cursor = nextCursor + 1
			continue
		}
		rootinsts = append(rootinsts, inst)
		b.cursor++
	}

	// Where each scope lands is known only now, since it depends on how many there are: the
	// dispatcher block comes first and every entry of it is the same size. So they are
	// written again, this time with the address they will have.
	// The runtime opens by saying where the first frame begins, and the scopes come after
	// the dispatcher, so both are ahead of every body.
	referenced := FRAME_START_SIZE
	if len(dispatchers) > 0 {
		referenced += DISPATCHER_BYTES_SIZE*len(dispatchers) + NO_MATCH_DISPATCHER_SIZE
	}
	for at := range dispatchers {
		d := &dispatchers[at]
		code := bytes.NewBuffer(make([]byte, 0))
		if err := WriteScope(code, d.Body, b.tapeSize, referenced+d.Offset); err != nil {
			return nil, err
		}
		d.Code = code
	}

	if len(rootinsts) > 0 {
		rootinsts = Lowering(rootinsts, b.tapeSize)
		root := bytes.NewBuffer(make([]byte, 0))
		// Code no scope holds: nobody called it, so its return ends the call.
		if _, err := WriteCode(root, NewIdentManagerAt(FrameNamesAt(rootinsts)), rootinsts, b.tapeSize, referenced+offset, true); err != nil {
			return nil, err
		}
		return &RuntimeCode{Root: root, Dispatchers: dispatchers}, nil
	}

	return &RuntimeCode{Dispatchers: dispatchers}, nil
}

func (b *Builder) WriteRuntimeBlock(bs io.Writer, rc *RuntimeCode) (int, error) {
	if _, err := WriteDispatchers(bs, rc.Dispatchers); err != nil {
		return 0, err
	}

	return WriteBodyCode(bs, rc.Dispatchers, rc.Root)
}

// Build assembles the program into bytecode and returns it.
//
// It returns the bytes rather than writing them: a phase takes values and gives values
// back, and deciding where bytecode lands — a file, a deployment, a test — belongs to
// whoever asked for it.
func (b *Builder) Build() ([]byte, error) {
	rc, err := b.PickRuntimeCode()
	if err != nil {
		return nil, err
	}

	out := bytes.NewBuffer(make([]byte, 0))

	runtimeSize := GetRuntimeCodeLength(rc)
	if runtimeSize > MAX_CONTRACT_SIZE {
		return nil, fmt.Errorf("the runtime is %d bytes and a chain keeps at most %d: this program cannot be deployed", runtimeSize, MAX_CONTRACT_SIZE)
	}

	if _, err := WriteInstantiateBlock(out, runtimeSize); err != nil {
		return nil, err
	}

	if _, err := b.WriteRuntimeBlock(out, rc); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

type NewBuilderOptions struct {
	// TapeSize is the width in bytes of every value. Zero means the default (8).
	TapeSize int
}

func NewBuilder(insts []ir.Instruction, options NewBuilderOptions) *Builder {
	return &Builder{
		tapeSize: byteutil.TapeSize(options.TapeSize),
		operands: make([][]byte, 0),
		cursor:   0,
		insts:    insts,
	}
}
