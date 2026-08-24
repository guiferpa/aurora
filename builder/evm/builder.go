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

// PickDeferAtCursor answers the scope written at this cursor, if one is: the instructions of
// its body and the name it was bound to.
//
// It finds the scope and does not write it. Where a scope lands depends on how many come
// before it, and none of that is known while they are still being found — so writing here
// meant writing every scope twice, throwing the first away, and reporting a failure to write
// as "there is no scope here", which is a different thing and hid the reason.
//
// A scope reaches the chain as an entry in the dispatcher, and what the dispatcher matches on
// is the name it was bound to. So the binding that follows the body is part of finding one: a
// scope nothing was bound to is nothing a transaction can reach.
func (b *Builder) PickDeferAtCursor(cursor int) (d *Dispatcher, nextCursor int, ok bool) {
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
	if end >= len(b.insts) {
		return nil, cursor, false
	}

	selectorInst := b.insts[end]
	if selectorInst.GetOpCode() != ir.OpIdent {
		return nil, cursor, false
	}

	body := Lowering(withoutNestedScopes(b.insts[cursor+1:end], b.tapeSize), b.tapeSize)
	return &Dispatcher{Selector: selectorInst.GetLeft().Bytes(), Body: body}, end, true
}

// withoutNestedScopes takes the scopes written inside this one out of its body.
//
// A scope's body is code that runs when the scope is called, never where it was written. Left
// in, the inner body was written straight into the outer one and ran on the way past: its
// return fired in the middle of code that had not asked for it, so
//
//	ident outer = defer { ident inner = defer { 1; }; 2; };
//
// answered 1 on chain where the program answers 2. It compiled, it deployed, and it said
// nothing — the worst way for a contract to be wrong.
//
// What replaces it is the neutral value, which is what the binding is worth here: a scope
// written inside another is not something a transaction can reach, and calling one is already
// refused, so on a chain the name it was bound to holds nothing. Replacing rather than dropping
// is what keeps the binding, and a scope whose last expression is one still answers what it
// answered.
func withoutNestedScopes(insts []ir.Instruction, tapeSize int) []ir.Instruction {
	out := make([]ir.Instruction, 0, len(insts))
	for at := 0; at < len(insts); at++ {
		inst := insts[at]
		if inst.GetOpCode() != ir.OpDefer {
			out = append(out, inst)
			continue
		}
		// A scope written inside this one carries how long its body is, and a scope inside
		// that one is inside those instructions, so stepping over them steps over every depth.
		out = append(out, ir.NewInstruction(
			inst.GetLabel(), ir.OpSave, ir.ImmOf(byteutil.FalseTape(tapeSize), tapeSize), ir.Nothing()))
		at += int(byteutil.ToUint64(inst.GetRight().Bytes()))
	}
	return out
}

// pickScopes separates the scopes a transaction can reach from the code that is not held by
// one, which is the top of the program.
func (b *Builder) pickScopes() (dispatchers []Dispatcher, root []ir.Instruction) {
	dispatchers = make([]Dispatcher, 0)
	root = make([]ir.Instruction, 0)

	for b.cursor < len(b.insts) {
		if d, nextCursor, ok := b.PickDeferAtCursor(b.cursor); ok {
			dispatchers = append(dispatchers, *d)
			// Skip the binding that named the scope: it is the dispatcher's selector and
			// has no meaning of its own on chain.
			b.cursor = nextCursor + 1
			continue
		}
		root = append(root, b.GetInstruction())
		b.cursor++
	}

	return dispatchers, root
}

// measureScopes answers how long each scope is and where it falls among them.
//
// It measures by writing to something that only counts, rather than by writing the whole scope
// into a buffer that is then thrown away — which is what this did, since where a scope lands
// is not known until every scope has been found and the scope has to be written again anyway.
func (b *Builder) measureScopes(dispatchers []Dispatcher, entries map[string]Entry) (int, error) {
	offset := 0
	for at := range dispatchers {
		d := &dispatchers[at]
		var measured counter
		if err := WriteScope(&measured, d.Body, b.tapeSize, 0, entries); err != nil {
			return 0, err
		}
		d.Offset, d.Length = offset, int(measured)
		offset += d.Length
	}
	return offset, nil
}

// PickRuntimeCode assembles the runtime in passes, each of which needs the one before it:
// the scopes are found, then measured, and only then written — a jump inside a scope carries
// an address in the contract, and that address depends on how many scopes come before it.
func (b *Builder) PickRuntimeCode() (*RuntimeCode, error) {
	dispatchers, rootinsts := b.pickScopes()

	// The names come before any address of one does: a call inside a scope has to be written
	// while the scopes are still being measured, and what it needs then is only that the name
	// it calls is a scope of this contract. The addresses are filled in below, into this same
	// map, once there are addresses to fill in.
	entries := make(map[string]Entry, len(dispatchers))
	for at := range dispatchers {
		d := &dispatchers[at]
		entries[string(d.Selector)] = Entry{Reads: FrameNamesAt(d.Body) / MEMORY_SLOT_SIZE}
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
		internal, err := ScopeInternalAt(referenced+d.Offset, d.Body, b.tapeSize)
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
		if err := WriteScope(code, d.Body, b.tapeSize, referenced+d.Offset, entries); err != nil {
			return nil, err
		}
		d.Code = code
	}

	if len(rootinsts) > 0 {
		rootinsts = Lowering(rootinsts, b.tapeSize)
		root := bytes.NewBuffer(make([]byte, 0))
		// Code no scope holds: nobody called it, so its return ends the call.
		if _, err := WriteCode(root, NewIdentManagerAt(FrameNamesAt(rootinsts)), rootinsts, referenced+offset, ScopeOf(rootinsts, b.tapeSize, entries, true)); err != nil {
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
