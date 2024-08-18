package environ

import (
	"fmt"
	"io"

	"github.com/guiferpa/aurora/emitter"
)

type EnvironContext string

type Environ struct {
	table     map[string][]byte
	functions map[string][]emitter.Instruction
	segments  []string
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

func (env *Environ) Print(w io.Writer) {
	for k, v := range env.table {
		fmt.Printf("%s: %v\n", k, v)
	}
}

func New(previous *Environ) *Environ {
	table := make(map[string][]byte)
	segments := make([]string, 0)
	functions := make(map[string][]emitter.Instruction)
	return &Environ{table, functions, segments, previous}
}
