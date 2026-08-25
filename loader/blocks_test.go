package loader

import (
	"testing"

	"github.com/guiferpa/aurora/wire/ir"
)

// Where a file ends is where the next begins, and the last one ends by answering — so nothing
// stops it. It is a place rather than a count for the reason the whole IR is: a file is a run
// through blocks, and where it stops is one of them.
func TestStopsAtNamesTheNextFile(t *testing.T) {
	program := Program{Ranges: []Range{{Top: 0}, {Top: 4}, {Top: 9}}}

	stops := program.StopsAt(0)
	if stops == nil {
		t.Fatal("the first file stops nowhere, and two follow it")
	}
	if !stops(ir.Point{Block: 4}) {
		t.Error("it does not stop where the second file begins")
	}
	if stops(ir.Point{Block: 9}) {
		t.Error("it stops where the third file begins, which is not where it ends")
	}
	if stops(ir.Point{Block: 4, At: 1}) {
		t.Error("it stops inside the second file rather than at the top of it")
	}

	if program.StopsAt(2) != nil {
		t.Error("the last file stops somewhere, and nothing follows it")
	}
}
