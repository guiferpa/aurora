package ir

import "github.com/guiferpa/aurora/wire/diag"

// Expression is one top-level expression of a program: where it begins among the blocks, and
// the label the value it returns ends up under.
//
// A caller that wants to report values in source order needs it. Reading what a program left
// behind after the whole of it has run cannot give that order: what print wrote along the way
// is already out, and the values would all come after it.
type Expression struct {
	At    Point
	Label []byte
}

// Program is what a source file compiled to: the blocks, where each top-level expression
// begins among them, and anything worth saying that did not stop the compilation.
//
// It used to carry the instructions as a list beside this, and the list was what said where
// control goes: an "if" carried how many instructions to skip, a scope's body how long it was.
// Every consumer worked that structure out again by counting, each in its own way, and two
// ways of counting one thing is two chances to count it differently. Nothing counts now.
type Program struct {
	Blocks      []Block
	Expressions []Expression
	Warnings    []diag.Warning
}

// A Label names the value an instruction leaves behind, so a later instruction can read it.
type Label []byte
