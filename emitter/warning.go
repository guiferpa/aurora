package emitter

import (
	"fmt"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/diag"
)

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
func checkDeferCapacity(nodes []ast.Node, tapeSize int) []diag.Warning {
	tapeSize = byteutil.TapeSize(tapeSize)
	if tapeSize >= 8 {
		// A scope would need more than 2^64 defers to wrap; nothing to say.
		return nil
	}
	capacity := uint64(1) << (8 * tapeSize)

	warnings := make([]diag.Warning, 0)
	var walk func(scope []ast.Node)
	walk = func(scope []ast.Node) {
		defers := 0
		for _, node := range scope {
			if countDefers(node, &defers, walk) {
				continue
			}
		}
		if uint64(defers) > capacity {
			warnings = append(warnings, diag.Warning{Message: fmt.Sprintf(
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
func checkAsserts(nodes []ast.Node) []diag.Warning {
	warnings := make([]diag.Warning, 0)

	var walk func(scope []ast.Node)
	walk = func(scope []ast.Node) {
		for _, node := range scope {
			if assertion, ok := node.(ast.AssertStatement); ok {
				warning := diag.Warning{Message: "assert only runs under 'aurora test'; ignored here"}
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
func childScopesOf(node ast.Node) []ast.Node {
	switch n := node.(type) {
	case ast.IdentLiteral:
		return []ast.Node{n.Value}
	case ast.BlockExpression:
		return n.Body
	case ast.DeferExpression:
		return n.Block.Body
	case ast.IfExpression:
		body := make([]ast.Node, 0, len(n.Body)+1)
		body = append(body, n.Body...)
		if n.Else != nil {
			body = append(body, n.Else.Body...)
		}
		return body
	case ast.PrintStatement:
		return []ast.Node{n.Param}
	case ast.ShapeLiteral:
		// A field can hold a deferred scope, so the values of a shape are walked like any
		// other place a scope can hide.
		return n.Values
	case ast.FieldExpression:
		return []ast.Node{n.Expression}
	case ast.ShapedExpression:
		return []ast.Node{n.Expression}
	default:
		return nil
	}
}

// countDefers adds node's own defers to count and walks the scopes it opens, since each of
// those keeps its own tally.
func countDefers(node ast.Node, count *int, walk func([]ast.Node)) bool {
	switch n := node.(type) {
	case ast.DeferExpression:
		*count++
		walk(n.Block.Body)
	case ast.IdentLiteral:
		return countDefers(n.Value, count, walk)
	case ast.BlockExpression:
		walk(n.Body)
	case ast.IfExpression:
		walk(n.Body)
		if n.Else != nil {
			walk(n.Else.Body)
		}
	case ast.PrintStatement:
		return countDefers(n.Param, count, walk)
	case ast.ShapeLiteral:
		for _, value := range n.Values {
			if !countDefers(value, count, walk) {
				return false
			}
		}
	case ast.FieldExpression:
		return countDefers(n.Expression, count, walk)
	case ast.ShapedExpression:
		return countDefers(n.Expression, count, walk)
	}
	return true
}
