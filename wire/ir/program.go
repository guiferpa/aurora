package ir

import "github.com/guiferpa/aurora/wire/diag"

// Expression is one top-level expression of a program: the range of instructions it
// compiled to, and the label the temp holding its value ends up under.
//
// A caller that wants to report values in source order needs this. Reading the temp map
// after the whole program has run cannot give it: the map has no order, and everything
// written by print along the way has already been emitted.
type Expression struct {
	From  int // index of its first instruction
	To    int // index one past its last instruction
	Label []byte
}

// Program is what a source file compiled to: the instruction stream, where each top-level
// expression sits inside it, and anything worth saying that did not stop the compilation.
type Program struct {
	Instructions []Instruction
	Expressions  []Expression
	Warnings     []diag.Warning
}

// A Label names the value an instruction leaves behind, so a later instruction can read it.
type Label []byte
