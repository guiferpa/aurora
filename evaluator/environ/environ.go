package environ

type Environ struct {
	args   map[uint64][]byte
	idents map[string][]byte
	defers map[string][]byte
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
	if len(value) > 0 {
		e.temps[key] = value
	}
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

func (e *Environ) SetDefer(key string, value []byte) {
	e.defers[key] = value
}

func (e *Environ) GetDefer(key string) []byte {
	return e.defers[key]
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
	Idents map[string][]byte
	Args   []byte
	Prev   *Environ
}

func NewEnviron(opts NewEnvironOptions) *Environ {
	args := make(map[uint64][]byte, 0)
	for i := 0; i < len(opts.Args); i += 32 {
		args[uint64(i/32)] = opts.Args[i : i+32]
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
