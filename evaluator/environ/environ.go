package environ

import "github.com/guiferpa/aurora/byteutil"

// calldataSlotSize is the ABI word used to pass arguments in, independent of the tape size.
const calldataSlotSize = 32

type Environ struct {
	args   map[uint64][]byte
	idents map[string][]byte
	defers map[string][]byte // key = hex(len at store time), value = blob (from, to, returnKey)
	temps  map[string][]byte
	prev   *Environ
}

func (e *Environ) Ahead(next *Environ) *Environ {
	next.prev = e
	return next
}

func (e *Environ) GetPrevious() *Environ {
	return e.prev
}

func (e *Environ) SetTemp(key string, value []byte) {
	e.temps[key] = value
}

func (e *Environ) GetTemp(key string) []byte {
	t := e.temps[key]
	delete(e.temps, key)
	return t
}

func (e *Environ) GetTemps() map[string][]byte {
	return e.temps
}

func (e *Environ) ClearTemps() {
	e.temps = make(map[string][]byte, 0)
}

func (e *Environ) SetIdent(key string, value []byte) {
	e.idents[key] = value
}

func (e *Environ) GetIdent(key string) []byte {
	curr := e
	for curr != nil {
		if c, ok := curr.idents[key]; ok {
			return c
		}
		curr = curr.prev
	}
	return nil
}

func (e *Environ) GetLocalIdent(key string) []byte {
	return e.idents[key]
}

// DefersLength returns the number of defers in this environ (used to build the next incremental key).
func (e *Environ) DefersLength() int {
	return len(e.defers)
}

func (e *Environ) SetDefer(key string, blob []byte) {
	if len(blob) > 0 {
		e.defers[key] = blob
	}
}

// GetDefer returns the defer blob for key, walking the environ chain (inner to outer).
func (e *Environ) GetDefer(key string) []byte {
	curr := e
	for curr != nil {
		if b, ok := curr.defers[key]; ok {
			return b
		}
		curr = curr.prev
	}
	return nil
}

func (e *Environ) SetArgument(key uint64, value []byte) {
	e.args[key] = value
}

func (e *Environ) SetArguments(args map[uint64][]byte) {
	e.args = args
}

func (e *Environ) GetArgument(key uint64) []byte {
	if arg, ok := e.args[key]; ok {
		return arg
	}
	return nil
}

func (e *Environ) GetArguments() map[uint64][]byte {
	return e.args
}

func (e *Environ) GetArgumentsLength() uint64 {
	return uint64(len(e.args))
}

type NewEnvironOptions struct {
	Idents   map[string][]byte
	Args     []byte
	Prev     *Environ
	TapeSize int
}

func NewEnviron(opts NewEnvironOptions) *Environ {
	// Arguments arrive as 32-byte ABI words (EVM calldata convention) and are narrowed to
	// tapes, so arguments(n) yields a value of the same width as everything else.
	args := make(map[uint64][]byte, 0)
	for i := 0; i < len(opts.Args); i += calldataSlotSize {
		args[uint64(i/calldataSlotSize)] = byteutil.PaddingTape(opts.Args[i:i+calldataSlotSize], opts.TapeSize)
	}
	idents := make(map[string][]byte, 0)
	if opts.Idents != nil {
		idents = opts.Idents
	}
	return &Environ{
		args:   args,
		idents: idents,
		defers: make(map[string][]byte),
		temps:  make(map[string][]byte),
		prev:   opts.Prev,
	}
}
