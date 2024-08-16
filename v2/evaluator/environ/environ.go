package environ

import (
	"fmt"
	"io"
)

type EnvironContext string

type Environ struct {
	table    map[string][]byte
	previous *Environ
}

func (env *Environ) Set(key string, value []byte) {
	env.table[key] = value
}

func (env *Environ) Get(key string) []byte {
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
	return &Environ{make(map[string][]byte), previous}
}
