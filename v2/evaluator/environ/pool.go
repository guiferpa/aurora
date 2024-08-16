package environ

type Pool struct {
	env *Environ
}

func (p *Pool) IsEmpty() bool {
	return p.env == nil
}

func (p *Pool) Set(key string, value []byte) {
	if curr := p.Current(); curr != nil {
		curr.Set(key, value)
	}
}

func (p *Pool) Query(key string) []byte {
	curr := p.env
	for curr != nil {
		if c := curr.Get(key); c != nil {
			return c
		}
		curr = curr.previous
	}
	return nil
}

func (p *Pool) Ahead() {
	p.env = New(p.env)
}

func (p *Pool) Back() {
	if !p.IsEmpty() {
		p.env = p.env.previous
	}
}

func (p *Pool) Current() *Environ {
	return p.env
}

func NewPool(env *Environ) *Pool {
	return &Pool{env}
}
