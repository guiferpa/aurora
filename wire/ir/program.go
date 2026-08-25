package ir

import "github.com/guiferpa/aurora/wire/diag"

// Expression is one top-level expression of a program: the range of instructions it
// compiled to, and the label the temp holding its value ends up under.
//
// A caller that wants to report values in source order needs this. Reading the temp map
// after the whole program has run cannot give it: the map has no order, and everything
// written by print along the way has already been emitted.
type Expression struct {
	From int // index of its first instruction
	To   int // index one past its last instruction
	// At is where the expression begins among the blocks. A runner that answers where each
	// expression happens rather than all of them at the end runs from one of these to the
	// next, and it is two numbers rather than one because instructions that run one after
	// another are not written one after another.
	At    Point
	Label []byte
}

// Program is what a source file compiled to: the instruction stream, the same program as
// blocks, where each top-level expression sits inside the stream, and anything worth saying
// that did not stop the compilation.
//
// The two forms describe the same program, and both are here while its consumers cross from
// one to the other. The stream says structure by counting — how many instructions an "if"
// skips, how long a scope's body is — so every consumer works the structure out again, each in
// its own way, and two ways of counting one thing is two chances to count it differently. The
// blocks say it once.
type Program struct {
	Instructions []Instruction
	Blocks       []Block
	Expressions  []Expression
	Warnings     []diag.Warning
}

// A Label names the value an instruction leaves behind, so a later instruction can read it.
type Label []byte
