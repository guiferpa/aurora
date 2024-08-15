package environ

type Pool struct {
	envs []*Environ
}

func (p *Pool) IsEmpty() bool {
	return len(p.envs) == 0
}

func (p *Pool) Set(key string, claim Claim) {
	if !p.IsEmpty() {
		curr := p.envs[len(p.envs)-1]
		curr.Set(key, claim)
	}
}

func (p *Pool) Query(key string) Claim {
	cursor := len(p.envs) - 1
	curr := p.envs[cursor]
	for ; cursor >= 0; cursor-- {
		if c := curr.Get(key); c != nil {
			return c
		}
		curr = p.envs[cursor]
	}
	return nil
}

func (p *Pool) Pop() *Environ {
	if !p.IsEmpty() {
		i := len(p.envs) - 1
		t := p.envs[i]
		p.envs = p.envs[:i]
		return t
	}
	return nil
}

func (p *Pool) Append(env *Environ) {
	p.envs = append(p.envs, env)
}

func (p *Pool) Current() *Environ {
	if !p.IsEmpty() {
		return p.envs[len(p.envs)-1]
	}
	return nil
}

func NewPool() *Pool {
	environ := New()
	return &Pool{[]*Environ{environ}}
}
