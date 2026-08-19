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
	// modules is where a program's modules keep their names, indexed by the module's own
	// name. It is born on the environ every chain ends at, because a module's names belong
	// to the program and not to a scope, and every module environ is handed the same map so
	// that a body running anywhere can reach it.
	modules map[string]*Environ
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
	if home := e.Holder(key); home != nil {
		return home.idents[key]
	}
	return nil
}

// Holder answers the environ a name is bound in, from here outwards, and nil when nothing on
// the chain has it.
//
// Which environ it was is worth knowing on its own: a deferred scope is an index counted in
// the environ that created it, so whoever wants the body has to look for it where the name
// was found and nowhere else.
func (e *Environ) Holder(key string) *Environ {
	curr := e
	for curr != nil {
		if _, ok := curr.idents[key]; ok {
			return curr
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

// GetLocalDefer answers the deferred scope stored here, without walking outwards.
func (e *Environ) GetLocalDefer(key string) []byte {
	return e.defers[key]
}

// Module answers the environ a module keeps its names in, and nil when no module of that name
// ever ran.
func (e *Environ) Module(id string) *Environ {
	return e.index()[id]
}

// OpenModule answers it too, making it the first time it is asked for. Whoever runs a module's
// body is what calls this: what that body binds belongs to the module rather than to whatever
// scope happened to be open.
//
// The environ it makes stands on its own, with no chain behind it. A module sees what it
// declared and what it imported, and nothing of whoever imported it.
func (e *Environ) OpenModule(id string) *Environ {
	root := e.root()
	if root.modules == nil {
		root.modules = make(map[string]*Environ)
	}
	if existing, ok := root.modules[id]; ok {
		return existing
	}
	next := NewEnviron(NewEnvironOptions{})
	next.modules = root.modules
	root.modules[id] = next
	return next
}

// index answers the module index this environ can reach. A module body runs with its own
// environ at the head of the chain and a called scope runs at the head of its caller's, so
// either way the index is somewhere along it.
func (e *Environ) index() map[string]*Environ {
	curr := e
	for curr != nil {
		if curr.modules != nil {
			return curr.modules
		}
		curr = curr.prev
	}
	return nil
}

// root is the far end of the chain: the environ a program starts in, and where its modules
// are indexed.
func (e *Environ) root() *Environ {
	curr := e
	for curr.prev != nil {
		curr = curr.prev
	}
	return curr
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
