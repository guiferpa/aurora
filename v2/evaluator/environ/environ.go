package environ

import (
	"fmt"
	"io"
)

type EnvironContext string

type Environ struct {
	table map[string]Claim
}

func (env *Environ) Set(key string, claim Claim) {
	env.table[key] = claim
}

func (env *Environ) Get(key string) Claim {
	if c, ok := env.table[key]; ok {
		return c
	}
	return nil
}

func (env *Environ) Print(w io.Writer) {
	for k, v := range env.table {
		fmt.Printf("%s: %v\n", k, v.Bytes())
	}
}

func New() *Environ {
	return &Environ{make(map[string]Claim)}
}
