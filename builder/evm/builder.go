package evm

import (
	"bytes"
	"io"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
)

const (
	BYTE_SIZE                = 8
	DISPATCHER_BYTES_SIZE    = 15
	NO_MATCH_DISPATCHER_SIZE = 1
	CALLDATA_SLOT_READABLE   = 32
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

type IdentManager struct {
	offsetIdents map[string]byte
}

func (m *IdentManager) GetOffset(ident []byte) byte {
	return m.offsetIdents[string(ident)]
}

func (m *IdentManager) SetOffset(ident string, offset byte) {
	m.offsetIdents[ident] = offset
}

func (m *IdentManager) GetLength() uint {
	return uint(len(m.offsetIdents))
}

func NewIdentManager() *IdentManager {
	return &IdentManager{offsetIdents: make(map[string]byte)}
}

type Builder struct {
	cursor       int
	insts        []emitter.Instruction
	operands     [][]byte
	logger       *Logger
	identManager *IdentManager
}

func (b *Builder) GetInstruction() emitter.Instruction {
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
	if inst.GetOpCode() != emitter.OpDefer {
		return nil, cursor, false
	}

	// OpDefer layout: [OpDefer] [body of length N] [OpIdent]. Right operand = N (body length in instructions).
	bodylength := byteutil.ToUint64(inst.GetRight())
	end := cursor + 1 + int(bodylength)
	if end > len(b.insts) {
		return nil, cursor, false
	}
	body := b.insts[cursor+1 : end]

	// Emit EVM bytecode for the defer body (OpBeginScope, ...stmts..., OpReturn).
	code := bytes.NewBuffer(make([]byte, 0))
	if _, err := WriteCode(code, b.identManager, body); err != nil {
		return nil, cursor, false
	}

	// Defer must be assigned to an ident (e.g. "ident f = defer { ... }"); that OpIdent is the selector.
	if end >= len(b.insts) {
		return nil, cursor, false
	}
	selectorInst := b.insts[end]
	if selectorInst.GetOpCode() != emitter.OpIdent {
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
	rootinsts := make([]emitter.Instruction, 0)
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
		root := bytes.NewBuffer(make([]byte, 0))
		if _, err := WriteCode(root, b.identManager, rootinsts); err != nil {
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

func (b *Builder) Build(w io.Writer) (int, error) {
	rc, err := b.PickRuntimeCode()
	if err != nil {
		return 0, err
	}

	bs := bytes.NewBuffer(make([]byte, 0))
	out := io.MultiWriter(bs, w)

	if _, err := WriteInstantiateBlock(out, byte(GetRuntimeCodeLength(rc))); err != nil {
		return 0, err
	}

	if _, err := b.WriteRuntimeBlock(out, rc); err != nil {
		return 0, err
	}

	if err := b.logger.Scanln(bs.Bytes()); err != nil {
		return 0, err
	}

	return bs.Len(), nil
}

type NewBuilderOptions struct {
	EnableLogging bool
}

func NewBuilder(insts []emitter.Instruction, options NewBuilderOptions) *Builder {
	return &Builder{
		operands:     make([][]byte, 0),
		identManager: NewIdentManager(),
		cursor:       0,
		insts:        insts,
		logger:       NewLogger(options.EnableLogging),
	}
}
