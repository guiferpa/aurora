// Package diag holds what a phase has to say about a program without refusing it.
//
// A warning crosses from whoever found it to whoever shows it — from the emitter or the
// backend to a command line — so it belongs to neither. Before this, the emitter owned the
// type and the backend had one of its own, which meant a host converting between two shapes
// of the same idea to print them in one list.
package diag

// A Warning is something worth saying about a program that is not a reason to refuse it.
// Compilation carries on and the program runs.
//
// Line and Column are 1-based and zero when the warning is about the program as a whole
// rather than a place in it. That used to be every warning a backend produced, because the IR
// carried instructions and no trace of where they came from; instructions carry an origin now,
// so a backend points at a place whenever the emitter had one to give it.
type Warning struct {
	Message string
	Line    int
	Column  int
}

func (w Warning) String() string {
	return w.Message
}

// Positioned reports whether the warning points at a place in the source.
func (w Warning) Positioned() bool {
	return w.Line > 0
}
