package environ

import (
	"github.com/guiferpa/aurora/emitter"
)

type Pool struct {
	globals map[string][]byte
	locals  *Environ
}

func (p *Pool) IsEmpty() bool {
	return p.locals == nil
}

func (p *Pool) GetLocal(key string) []byte {
	if curr := p.Current(); curr != nil {
		return curr.GetLocal(key)
	}
	return nil
}

func (p *Pool) SetLocal(key string, value []byte) {
	if curr := p.Current(); curr != nil {
		curr.SetLocaL(key, value)
	}
}

func (p *Pool) QueryLocal(key string) []byte {
	curr := p.locals
	for curr != nil {
		if c := curr.GetLocal(key); c != nil {
			return c
		}
		curr = curr.previous
	}
	return nil
}

func (p *Pool) QueryFunctionSegment(key string) *FunctionSegment {
	curr := p.locals
	for curr != nil {
		if s := curr.GetSegment(key); s != nil {
			return s
		}
		curr = curr.previous
	}
	return nil
}

func (p *Pool) SetContext(cursor uint64, insts []emitter.Instruction) {
	if curr := p.Current(); curr != nil {
		curr.SetContext(NewContext(cursor, insts))
	}
}

func (p *Pool) GetContext() *Context {
	if curr := p.Current(); curr != nil {
		return curr.GetContext()
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
