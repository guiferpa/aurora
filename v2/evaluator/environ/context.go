package environ

import "github.com/guiferpa/aurora/emitter"

type Context struct {
	cursor int
	insts  []emitter.Instruction
}

func (ctx *Context) GetCursor() int {
	return ctx.cursor
}

func (ctx *Context) GetInstructions() []emitter.Instruction {
	return ctx.insts
}

func NewContext(cursor int, insts []emitter.Instruction) *Context {
	return &Context{cursor, insts}
}
