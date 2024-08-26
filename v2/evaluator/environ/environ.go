package environ

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/emitter"
)

type Environ struct {
	table    map[string][]byte
	segpool  map[string]*FunctionSegment
	ctx      *Context
	previous *Environ
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
  /*
	for i, inst := range insts {
		fmt.Println(fmt.Sprintf("segment(%s):", key), i, fmt.Sprintf("%x: %x %x %x", inst.GetLabel(), inst.GetOpCode(), inst.GetLeft(), inst.GetRight()))
	}
  */
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

func (env *Environ) Print(w io.Writer) {
	for k, v := range env.table {
		fmt.Printf("%s: %v\n", k, v)
	}
}

func New(previous *Environ) *Environ {
	table := make(map[string][]byte)
	segpool := make(map[string]*FunctionSegment, 0)
	return &Environ{table, segpool, nil, previous}
}
