package environ

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/evaluator/segment"
)

type EnvironContext string

type Environ struct {
	table     map[string][]byte
	segpool   map[string]*segment.Segment
	segcurr   string
	functions map[string][]emitter.Instruction
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

func (env *Environ) NoSegment() {
	env.segcurr = ""
}

func (env *Environ) SetSegment(key string) {
	env.segcurr = key
}

func (env *Environ) GetCurrentSegment() *segment.Segment {
	if len(env.segcurr) > 0 {
		return env.segpool[env.segcurr]
	}
	return nil
}

func (env *Environ) GetSegment(key string) *segment.Segment {
	return env.segpool[key]
}

func (env *Environ) Print(w io.Writer) {
	for k, v := range env.table {
		fmt.Printf("%s: %v\n", k, v)
	}
}

func New(previous *Environ) *Environ {
	table := make(map[string][]byte)
	segpool := make(map[string]*segment.Segment, 0)
	segcurr := ""
	functions := make(map[string][]emitter.Instruction)
	return &Environ{table, segpool, segcurr, functions, previous}
}
