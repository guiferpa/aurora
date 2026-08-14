package emitter

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/parser"
)

// A Warning is something worth saying about a program that is not a reason to refuse it.
// Compilation carries on and the program runs.
//
// Line and Column are 1-based and zero when the warning is about the program as a whole
// rather than a place in it.
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

// checkDeferCapacity reports scopes holding more deferred scopes than a tape can index.
//
// The value of a defer is its index as a tape, so a tape of N bytes can name 2^(8N) scopes
// — 256 of them when a tape is one byte. Past that the index wraps, and a call reaches a
// different scope instead of failing, which is the kind of thing a program should be told
// about before it runs.
//
// This is a warning rather than an error because the count is static: it cannot know how
// often a scope actually runs. And it is never checked at runtime — a program that is
// already running should not be stopped by a limit its shape implied.
func checkDeferCapacity(nodes []parser.Node, tapeSize int) []Warning {
	tapeSize = byteutil.TapeSize(tapeSize)
	if tapeSize >= 8 {
		// A scope would need more than 2^64 defers to wrap; nothing to say.
		return nil
	}
	capacity := uint64(1) << (8 * tapeSize)

	warnings := make([]Warning, 0)
	var walk func(scope []parser.Node)
	walk = func(scope []parser.Node) {
		defers := 0
		for _, node := range scope {
			if countDefers(node, &defers, walk) {
				continue
			}
		}
		if uint64(defers) > capacity {
			warnings = append(warnings, Warning{Message: fmt.Sprintf(
				"%d deferred scopes in one scope, but a %d-byte tape can only name %d: calls past that reach the wrong scope",
				defers, tapeSize, capacity)})
		}
	}
	walk(nodes)
	return warnings
}

// checkAsserts reports every assertion in the tree.
//
// An assertion belongs to "aurora test"; a plain run consumes it and moves on, so the
// point of the warning is to say that out loud rather than let someone believe a check
// ran. Each one is named by position, so an editor can jump to it.
func checkAsserts(nodes []parser.Node) []Warning {
	warnings := make([]Warning, 0)

	var walk func(scope []parser.Node)
	walk = func(scope []parser.Node) {
		for _, node := range scope {
			if assertion, ok := node.(parser.AssertStatement); ok {
				warning := Warning{Message: "assert only runs under 'aurora test'; ignored here"}
				if assertion.Token != nil {
					warning.Line = assertion.Token.GetLine()
					warning.Column = assertion.Token.GetColumn()
				}
				warnings = append(warnings, warning)
				continue
			}
			walk(childScopesOf(node))
		}
	}
	walk(nodes)

	return warnings
}

// childScopesOf returns the expressions a node holds, so a walk reaches what is nested.
func childScopesOf(node parser.Node) []parser.Node {
	switch n := node.(type) {
	case parser.IdentLiteral:
		return []parser.Node{n.Value}
	case parser.BlockExpression:
		return n.Body
	case parser.DeferExpression:
		return n.Block.Body
	case parser.IfExpression:
		body := make([]parser.Node, 0, len(n.Body)+1)
		body = append(body, n.Body...)
		if n.Else != nil {
			body = append(body, n.Else.Body...)
		}
		return body
	case parser.PrintStatement:
		return []parser.Node{n.Param}
	default:
		return nil
	}
}

// countDefers adds node's own defers to count and walks the scopes it opens, since each of
// those keeps its own tally.
func countDefers(node parser.Node, count *int, walk func([]parser.Node)) bool {
	switch n := node.(type) {
	case parser.DeferExpression:
		*count++
		walk(n.Block.Body)
	case parser.IdentLiteral:
		return countDefers(n.Value, count, walk)
	case parser.BlockExpression:
		walk(n.Body)
	case parser.IfExpression:
		walk(n.Body)
		if n.Else != nil {
			walk(n.Else.Body)
		}
	case parser.PrintStatement:
		return countDefers(n.Param, count, walk)
	}
	return true
}
