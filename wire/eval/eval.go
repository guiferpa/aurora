// Package eval holds what evaluating a program answers with.
//
// The evaluator is the end of the pipeline, and what it produces crosses to a host just like
// tokens, a tree and IR cross between phases: the value of every expression, and what became
// of every assertion. Those are the artefacts of a run, so they live here rather than inside
// whoever produced them — "aurora test" reporting on assertions had to name the evaluator to
// do it.
//
// It is named after the phase that produces it, the way wire/ir is named after what it holds.
// Not "run": in Aurora a run is a run of tapes, which is a value, so the word is taken — two
// functions of the evaluator already use it for exactly that.
package eval

// Returns is the value each expression answered with, by the label the IR gave it.
//
// It is keyed by label and not by position because a label is what an instruction carries: a
// caller asking what one expression is worth asks for its label, and one that ran a whole
// program walks the labels it emitted.
type Returns map[string][]byte

// AssertResult is one assertion and what became of it. A run collects every one, not only the
// failures, so a report can say how many held.
type AssertResult struct {
	Passed  bool
	Message string
}
