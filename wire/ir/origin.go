package ir

// An Origin is where in the source an instruction came from.
//
// It is metadata, in the strict sense: dropping it changes nothing about what a program does.
// What it changes is whether a phase after the emitter can point at a place. A backend that
// cannot say where means a warning that names a feature and not the line it was written on,
// and the person is left to search.
//
// The emitter is the only phase that can supply it. Every node of the tree carries the token
// it was parsed from, and until now the emitter held that and let it go.
//
// Zero means the emitter had nothing to point at — a value it invented rather than one
// somebody wrote, like the neutral tape a declaration returns. Line and Column are
// 1-based, matching diag.Warning and the token positions it comes from.
type Origin struct {
	Line   int
	Column int
}

// Known reports whether the origin points at a place in the source.
func (o Origin) Known() bool {
	return o.Line > 0
}
