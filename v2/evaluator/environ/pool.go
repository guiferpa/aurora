package environ

type Pool struct {
	globals map[string][]byte
	locals  *Environ
}

func (p *Pool) IsEmpty() bool {
	return p.locals == nil
}

func (p *Pool) SetLocal(key string, value []byte) {
	if curr := p.Current(); curr != nil {
		curr.SetLocaL(key, value)
	}
}

func (p *Pool) Query(key string) []byte {
	curr := p.locals
	for curr != nil {
		if c := curr.GetLocal(key); c != nil {
			return c
		}
		curr = curr.previous
	}
	return nil
}

func (p *Pool) Ahead() {
	p.locals = New(p.locals)
}

func (p *Pool) Back() {
	if !p.IsEmpty() {
		p.locals = p.locals.previous
	}
}

func (p *Pool) Current() *Environ {
	return p.locals
}

func NewPool(locals *Environ) *Pool {
	globals := make(map[string][]byte)
	return &Pool{globals, locals}
}
