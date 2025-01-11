package environ

import "github.com/guiferpa/aurora/emitter"

type Context struct {
	cursor uint64
	insts  []emitter.Instruction
}

func (ctx *Context) GetCursor() uint64 {
	return ctx.cursor
}

func (ctx *Context) GetInstructions() []emitter.Instruction {
	return ctx.insts
}

func NewContext(cursor uint64, insts []emitter.Instruction) *Context {
	return &Context{cursor, insts}
}
