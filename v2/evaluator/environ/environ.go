package environ

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/emitter"
)

type Environ struct {
	arguments map[uint64][]byte
	table     map[string][]byte
	segpool   map[string]*FunctionSegment
	ctx       *Context
	previous  *Environ
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

func (env *Environ) SetSegment(key string, insts []emitter.Instruction, begin, end uint64) {
	env.segpool[key] = &FunctionSegment{insts, begin, end}
}

func (env *Environ) GetSegment(key string) *FunctionSegment {
	return env.segpool[key]
}

func (env *Environ) SetContext(ctx *Context) {
	env.ctx = ctx
}

func (env *Environ) GetContext() *Context {
	return env.ctx
}

func (env *Environ) PushArgument(arg []byte) {
	index := uint64(len(env.arguments))
	env.arguments[index] = arg
}

func (env *Environ) GetArgument(index uint64) []byte {
	args := env.arguments
	if arg, ok := args[index]; ok {
		return arg
	}
	return nil
}

func (env *Environ) Print(w io.Writer) {
	for k, v := range env.table {
		fmt.Printf("%s: %v\n", k, v)
	}
}

func New(previous *Environ) *Environ {
	table := make(map[string][]byte)
	segpool := make(map[string]*FunctionSegment, 0)
	arguments := make(map[uint64][]byte, 0)
	return &Environ{arguments, table, segpool, nil, previous}
}
