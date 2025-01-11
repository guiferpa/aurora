package environ

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/emitter"
)

type Environ struct {
	args  map[uint64][]byte // Arguments for environment
	table map[string][]byte
	scs   map[string]*ScopeCallable // Segments of functions
	ctx   *Context
	prev  *Environ          // Previous environment
}

func (env *Environ) SetLocaL(key string, value []byte) {
	env.table[key] = value
}

func (env *Environ) GetLocal(key string) []byte {
	if c, ok := env.table[key]; ok {
		return c
	}
	return nil
}

func (env *Environ) SetScopeCallable(key string, insts []emitter.Instruction, begin, end uint64) {
	env.scs[key] = &ScopeCallable{insts, begin, end}
}

func (env *Environ) GetScopeCallable(key string) *ScopeCallable {
	return env.scs[key]
}

func (env *Environ) SetContext(ctx *Context) {
	env.ctx = ctx
}

func (env *Environ) GetContext() *Context {
	return env.ctx
}

func (env *Environ) PushArgument(arg []byte) {
	index := uint64(len(env.args))
	env.args[index] = arg
}

func (env *Environ) GetArgument(index uint64) []byte {
	args := env.args
	if arg, ok := args[index]; ok {
		return arg
	}
	return nil
}

func (env *Environ) PrintTable(w io.Writer) {
	for k, v := range env.table {
		fmt.Printf("%s: %v\n", k, v)
	}
}

func New(prev *Environ) *Environ {
	return &Environ{
		args:  make(map[uint64][]byte, 0),
		table: make(map[string][]byte, 0),
		scs:   make(map[string]*ScopeCallable, 0),
		ctx:   nil,
		prev:  prev,
	}
}
