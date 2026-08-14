package emitter

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/parser"
)

// A Warning is something worth saying about a program that is not a reason to refuse it.
// Compilation carries on and the program runs.
type Warning struct {
	Message string
}

func (w Warning) String() string {
	return w.Message
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
	case parser.EchoStatement:
		return countDefers(n.Param, count, walk)
	}
	return true
}
