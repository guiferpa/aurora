// Package evm compiles the emitter's IR to EVM bytecode.
//
// Architecture: the Builder is intended to be mechanical (emit opcodes, resolve offsets).
// Stack order and LIFO semantics should be guaranteed by a separate Lowering phase that
// consumes IR and produces a stream already in EVM stack order. See
// docs/compiler_pipeline_and_lowering.md for the target pipeline (IR → Lowering → Builder).
package evm

import (
	"bytes"
	"io"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ir"
)

const (
	BYTE_SIZE                = 8
	DISPATCHER_BYTES_SIZE    = 15
	NO_MATCH_DISPATCHER_SIZE = 1
	CALLDATA_SLOT_READABLE   = 32
	MEMORY_SLOT_SIZE         = 32
	INSTANTIATE_BLOCK_SIZE   = 12
)

type Dispatcher struct {
	Selector []byte
	Offset   int
	Length   int
	Code     *bytes.Buffer
}

type RuntimeCode struct {
	Root        *bytes.Buffer
	Dispatchers []Dispatcher
}

type Builder struct {
	tapeSize     int
	cursor       int
	insts        []ir.Instruction
	operands     [][]byte
	identManager *IdentManager
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
	bodylength := byteutil.ToUint64(inst.GetRight())
	end := cursor + 1 + int(bodylength)
	if end > len(b.insts) {
		return nil, cursor, false
	}
	body := b.insts[cursor+1 : end]
	body = Lowering(body)

	// Emit EVM bytecode for the defer body (OpBeginScope, ...exprs..., OpReturn).
	code := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteCode(code, b.identManager, body, b.tapeSize); err != nil {
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
	selector := selectorInst.GetLeft()

	// Prepend OpJumpDestiny so the EVM can jump to this block when the selector matches.
	d = &Dispatcher{
		Selector: selector,
		Code:     bytes.NewBuffer(append([]byte{OpJumpDestiny}, code.Bytes()...)),
		Offset:   offset,
		Length:   code.Len(),
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
			offset += 1 + d.Length
			// Skip the OpIdent that assigns the defer to a variable; it has no EVM meaning (selector is already in the dispatcher).
			b.cursor = nextCursor + 1
			continue
		}
		rootinsts = append(rootinsts, inst)
		b.cursor++
	}

	if len(rootinsts) > 0 {
		rootinsts = Lowering(rootinsts)
		root := bytes.NewBuffer(make([]byte, 0))
		if _, err := WriteCode(root, b.identManager, rootinsts, b.tapeSize); err != nil {
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

	if _, err := WriteInstantiateBlock(out, byte(GetRuntimeCodeLength(rc))); err != nil {
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
		tapeSize:     byteutil.TapeSize(options.TapeSize),
		operands:     make([][]byte, 0),
		identManager: NewIdentManager(),
		cursor:       0,
		insts:        insts,
	}
}
