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
	// TOP_BLOCK is the block the program begins at. Everything a transaction can reach is
	// named from it, and it is the code that runs when a contract has no scope to dispatch to.
	TOP_BLOCK = 0
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
	// RETURN_TO_CHAIN_SIZE is what handing an ordinary value to the chain measures. A run
	// measures something else, and ReturnToChainSize answers for both by writing it.
)

type Dispatcher struct {
	Selector []byte
	Offset   int
	Length   int
	Code     *bytes.Buffer
	// Entry is the block the scope begins at, kept so it can be written again once where it
	// lands is known.
	Entry ir.BlockID
}

type RuntimeCode struct {
	Root        *bytes.Buffer
	Dispatchers []Dispatcher
}

type Builder struct {
	tapeSize int
	blocks   []ir.Block
	// problems is what was wrong with the IR it was given, kept from when it arrived so the
	// answer does not depend on what the lowering made of it.
	problems []ir.Problem
}

// PickScopes answers the scopes a transaction can reach, and the block that is the top of the
// program.
//
// A scope reaches the chain as an entry in the dispatcher, and what the dispatcher matches on
// is the name it was bound to — so a scope is found by finding the binding that named it. That
// is one pass over one block, where it used to be a walk that read a length off an instruction
// and sliced the list, and did so only at the top of it.
func (b *Builder) PickScopes() []Dispatcher {
	dispatchers := make([]Dispatcher, 0)
	if len(b.blocks) == 0 {
		return dispatchers
	}

	// Every block the program runs through, and not only the first: a program made of several
	// files is one run through all of them, so a scope bound in the second is as reachable by
	// a transaction as one bound in the first.
	// A scope is worth the block its body became, and the binding that names it reads that
	// value like any other. So finding one is finding the two together: what a scope is worth,
	// and the name it was bound to.
	scopes := make(map[string]ir.BlockID)
	for _, id := range ir.Reaches(b.blocks, TOP_BLOCK) {
		for _, inst := range b.blocks[id].Insts {
			if inst.GetOpCode() == ir.OpSave && inst.GetLeft().Kind() == ir.KindBlock {
				scopes[byteutil.ToHex(inst.GetLabel())] = inst.GetLeft().Block()
				continue
			}
			if inst.GetOpCode() != ir.OpIdent {
				continue
			}
			if entry, named := scopes[byteutil.ToHex(inst.GetRight().Bytes())]; named {
				dispatchers = append(dispatchers, Dispatcher{
					Selector: inst.GetLeft().Bytes(),
					Entry:    entry,
				})
			}
		}
	}
	return dispatchers
}

// measureScopes answers how long each scope is and where it falls among them.
//
// It measures by writing to something that only counts, rather than by writing the whole scope
// into a buffer that is then thrown away — where a scope lands is not known until every scope
// has been found, so it has to be written again anyway.
func (b *Builder) measureScopes(dispatchers []Dispatcher, entries map[string]Entry) (int, error) {
	offset := 0
	for at := range dispatchers {
		d := &dispatchers[at]
		var measured counter
		if err := WriteScope(&measured, b.blocks, d.Entry, b.tapeSize, 0, entries); err != nil {
			return 0, err
		}
		d.Offset, d.Length = offset, int(measured)
		offset += d.Length
	}
	return offset, nil
}

// PickRuntimeCode assembles the runtime in passes, each of which needs the one before it: the
// scopes are found, then measured, and only then written — a jump inside a scope carries an
// address in the contract, and that address depends on how many scopes come before it.
func (b *Builder) PickRuntimeCode() (*RuntimeCode, error) {
	dispatchers := b.PickScopes()

	// The names come before any address of one does: a call inside a scope has to be written
	// while the scopes are still being measured, and what it needs then is only that the name
	// it calls is a scope of this contract, and how many values that scope takes. The
	// addresses are filled into this same map below, once there are addresses to fill in.
	entries := make(map[string]Entry, len(dispatchers))
	for at := range dispatchers {
		d := &dispatchers[at]
		entries[string(d.Selector)] = Entry{Reads: len(b.blocks[d.Entry].Params)}
	}

	offset, err := b.measureScopes(dispatchers, entries)
	if err != nil {
		return nil, err
	}

	// The runtime opens by saying where the first frame begins, and the scopes come after the
	// dispatcher, so both are ahead of every body.
	referenced := FRAME_START_SIZE
	if len(dispatchers) > 0 {
		referenced += DISPATCHER_BYTES_SIZE*len(dispatchers) + NO_MATCH_DISPATCHER_SIZE
	}

	for at := range dispatchers {
		d := &dispatchers[at]
		internal, err := ScopeInternalAt(referenced+d.Offset, len(b.blocks[d.Entry].Params), b.tapeSize, b.blocks[d.Entry].Tapes)
		if err != nil {
			return nil, err
		}
		entry := entries[string(d.Selector)]
		entry.At = internal
		entries[string(d.Selector)] = entry
	}

	for at := range dispatchers {
		d := &dispatchers[at]
		code := bytes.NewBuffer(make([]byte, 0))
		if err := WriteScope(code, b.blocks, d.Entry, b.tapeSize, referenced+d.Offset, entries); err != nil {
			return nil, err
		}
		d.Code = code
	}

	// Code no scope holds: nobody called it, so ending it ends the call. Its blocks are the
	// ones the top of the program reaches, and a scope is not among them — a binding names a
	// block, and naming is not going.
	top := layoutOf(b.blocks, TOP_BLOCK)
	root := bytes.NewBuffer(make([]byte, 0))
	if err := writeBlocks(root, b.blocks, top, referenced+offset,
		ScopeOf(b.blocks, top, b.tapeSize, entries, true)); err != nil {
		return nil, err
	}

	return &RuntimeCode{Root: root, Dispatchers: dispatchers}, nil
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
	if len(b.problems) > 0 {
		return nil, fmt.Errorf("this program does not say what it means, and no bytecode was written: %w", b.problems[0])
	}

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

func NewBuilder(blocks []ir.Block, options NewBuilderOptions) *Builder {
	tapeSize := byteutil.TapeSize(options.TapeSize)

	// Checked as it arrives, before anything here touches it. The whole failure mode of this
	// backend was a binary that came out, deployed, answered and was quietly wrong, and what
	// kept that from happening was somebody keeping a list of opcodes in agreement with the
	// emitter — which is not an assertion.
	//
	// It is the IR that is checked and not what the lowering makes of it. What comes out of
	// the lowering is stack-scheduled and no longer a plain program: a binding that a scope
	// ends with leaves nothing on the stack, so the lowering writes a push under that same
	// name to stand in for it. Two values under one name is exactly what Verify refuses, and
	// it is right to — of the IR. Of a schedule for one machine it is not a question.
	problems := ir.Verify(blocks)

	// Every block is put in the order the stack needs before any of it is written. It used to
	// be done a scope at a time, as each was found, which was the same work spread out — and
	// it had to be, because a scope was a stretch of a list and the list was walked once.
	lowered := make([]ir.Block, len(blocks))
	for at, block := range blocks {
		lowered[at] = LowerBlock(block, tapeSize)
	}

	return &Builder{tapeSize: tapeSize, blocks: lowered, problems: problems}
}
