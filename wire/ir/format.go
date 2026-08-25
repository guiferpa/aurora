package ir

import (
	"bytes"
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
)

// Format renders instructions one per line, which is how the IR is read by a person.
//
// It lives with the IR rather than with whoever shows it: writing the vocabulary down is part
// of the vocabulary, the way a token knows how to spell itself. What belongs to a host is the
// decision to show it and the writer it goes to.
//
// It returns the text rather than printing it: a package that writes takes that decision away
// — and it also gets in the way of a test, which wants the string to compare.
func Format(insts []Instruction) string {
	bs := bytes.NewBuffer(make([]byte, 0))
	for _, inst := range insts {
		fmt.Fprintf(bs, "%s %s", byteutil.ToHexPretty(inst.GetLabel()), ResolveOpCode(inst.GetOpCode()))
		// Every operand, not the first two: an instruction over a run has as many as the
		// construction it came from, and printing a pair of them would be a line that reads
		// as complete and is not.
		for _, operand := range inst.GetOperands() {
			fmt.Fprintf(bs, " %s", operand)
		}
		fmt.Fprintln(bs)
	}
	return bs.String()
}

// FormatBlocks writes a program as a person reads it: each block, what it takes, what it
// computes, and how it ends.
//
// It replaces printing a list. A list read in order was once the program in order, and it is
// not: the block a run goes to next is the one its terminator names, which is what these lines
// say and a list never could.
func FormatBlocks(blocks []Block) string {
	bs := bytes.NewBuffer(make([]byte, 0))
	for _, block := range blocks {
		fmt.Fprintf(bs, "b%d(", block.ID)
		for at, param := range block.Params {
			if at > 0 {
				fmt.Fprint(bs, ", ")
			}
			if len(param) == 0 {
				fmt.Fprint(bs, "_")
				continue
			}
			fmt.Fprint(bs, byteutil.ToHexPretty(param))
		}
		fmt.Fprintln(bs, ")")

		for _, inst := range block.Insts {
			fmt.Fprintf(bs, "    %s %s", byteutil.ToHexPretty(inst.GetLabel()), ResolveOpCode(inst.GetOpCode()))
			for _, operand := range inst.GetOperands() {
				fmt.Fprintf(bs, " %s", operand)
			}
			fmt.Fprintln(bs)
		}

		fmt.Fprintf(bs, "    %s\n", block.Term)
	}
	return bs.String()
}

// String writes how a block ends, which is the one thing that says where a program goes next.
func (t Terminator) String() string {
	switch t.Kind {
	case Ret:
		return fmt.Sprintf("ret %s", t.Value)
	case Br:
		return fmt.Sprintf("br %s", t.Targets[0])
	case BrIf:
		return fmt.Sprintf("brif %s -> %s, %s", t.Cond, t.Targets[0], t.Targets[1])
	}
	return "?"
}

// String writes a block and what is handed to it.
func (t Target) String() string {
	bs := bytes.NewBuffer(make([]byte, 0))
	fmt.Fprintf(bs, "b%d(", t.Block)
	for at, arg := range t.Args {
		if at > 0 {
			fmt.Fprint(bs, ", ")
		}
		fmt.Fprint(bs, arg)
	}
	fmt.Fprint(bs, ")")
	return bs.String()
}
