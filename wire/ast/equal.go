// Comparing two trees is a question about the shape, so it travels with the shape: whoever
// asks it — a parser test today, anything else tomorrow — should not need the parser to do it.

package ast

import (
	"bytes"
	"reflect"
	"slices"

	"github.com/guiferpa/aurora/wire/token"
)

// Equal compares two trees by structure and by the tokens they carry, ignoring pointer
// identity. It travels with the tree: comparing two of them is a question about the shape,
// and whoever asks it — a parser test today, anything else tomorrow — should not need the
// parser to do it.
func Equal(got, want AST) bool {
	if got.Filename != want.Filename || len(got.Nodes) != len(want.Nodes) {
		return false
	}
	for i := range got.Nodes {
		if !nodeEqual(got.Nodes[i], want.Nodes[i]) {
			return false
		}
	}
	return true
}

// sameKind answers whether b is the kind of node a is, and whether the two then compare
// alike.
//
// Every case used to spell out the same three lines — assert the kind, give up if it is
// another one, compare — so the comparison, which is the only part that differed, was buried
// in the middle of twenty-one repetitions of the same shape. That is what carried nodeEqual
// to 107 of cognitive complexity, the highest in the project.
func sameKind[T Node](b Node, a T, alike func(a, b T) bool) bool {
	vb, ok := b.(T)
	return ok && alike(a, vb)
}

// nodeEqual answers whether two trees are the same tree. The kind of node picks how it is
// compared; a kind with no case falls back to a deep comparison, which is what keeps a node
// added tomorrow comparable before anyone writes one for it.
func nodeEqual(a, b Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch va := a.(type) {
	case NumberLiteral:
		return sameKind(b, va, numberEqual)
	case BooleanLiteral:
		return sameKind(b, va, booleanEqual)
	case BinaryExpression:
		return sameKind(b, va, binaryEqual)
	case IfExpression:
		return sameKind(b, va, ifEqual)
	case BlockExpression:
		return sameKind(b, va, blockEqual)
	case IdentLiteral:
		return sameKind(b, va, identEqual)
	case PrintStatement:
		return sameKind(b, va, printEqual)
	case AssertStatement:
		return sameKind(b, va, assertEqual)
	case UnaryExpression:
		return sameKind(b, va, unaryEqual)
	case StructDeclaration:
		return sameKind(b, va, structDeclarationEqual)
	case UseDeclaration:
		return sameKind(b, va, useDeclarationEqual)
	case StructLiteral:
		return sameKind(b, va, structLiteralEqual)
	case FieldExpression:
		return sameKind(b, va, fieldEqual)
	case ShapedExpression:
		return sameKind(b, va, shapedEqual)
	case DeferExpression:
		return sameKind(b, va, deferEqual)
	case CalleeLiteral:
		return sameKind(b, va, calleeEqual)
	case IdentifierLiteral:
		return sameKind(b, va, identifierEqual)
	case OperationLiteral:
		return sameKind(b, va, opEqual)
	case TapeBracketExpression:
		return sameKind(b, va, tapeEqual)
	case FeedExpression:
		return sameKind(b, va, feedEqual)
	case RelativeExpression:
		return sameKind(b, va, relativeEqual)
	case BooleanExpression:
		return sameKind(b, va, booleanExpressionEqual)
	default:
		return reflect.DeepEqual(a, b)
	}
}

func numberEqual(a, b NumberLiteral) bool {
	return a.Value == b.Value && token.Equal(a.Token, b.Token)
}

func booleanEqual(a, b BooleanLiteral) bool {
	return bytes.Equal(a.Value, b.Value) && token.Equal(a.Token, b.Token)
}

func opEqual(a, b OperationLiteral) bool {
	return a.Value == b.Value && token.Equal(a.Token, b.Token)
}

func identifierEqual(a, b IdentifierLiteral) bool {
	return a.Value == b.Value && token.Equal(a.Token, b.Token)
}

// The three expressions with two operands differ only in what they mean, and the operator is
// compared in all of them: it is the half of the tree that says which one it is.
func binaryEqual(a, b BinaryExpression) bool {
	return nodeEqual(a.Left, b.Left) && nodeEqual(a.Right, b.Right) && opEqual(a.Operation, b.Operation)
}

func relativeEqual(a, b RelativeExpression) bool {
	return nodeEqual(a.Left, b.Left) && nodeEqual(a.Right, b.Right) && opEqual(a.Operation, b.Operation)
}

func booleanExpressionEqual(a, b BooleanExpression) bool {
	return nodeEqual(a.Left, b.Left) && nodeEqual(a.Right, b.Right) && opEqual(a.Operation, b.Operation)
}

func unaryEqual(a, b UnaryExpression) bool {
	return nodeEqual(a.Expression, b.Expression) && opEqual(a.Operation, b.Operation)
}

// An if is equal to another when both take the same branch on the same test — so an else
// that only one of them has is already a difference, before its body is read.
func ifEqual(a, b IfExpression) bool {
	if !nodeEqual(a.Test, b.Test) || !nodesEqual(a.Body, b.Body) {
		return false
	}
	if (a.Else == nil) != (b.Else == nil) {
		return false
	}
	return a.Else == nil || elseEqual(*a.Else, *b.Else)
}

func blockEqual(a, b BlockExpression) bool {
	return nodesEqual(a.Body, b.Body)
}

func elseEqual(a, b ElseExpression) bool {
	return nodesEqual(a.Body, b.Body)
}

func deferEqual(a, b DeferExpression) bool {
	return blockEqual(a.Block, b.Block)
}

func identEqual(a, b IdentLiteral) bool {
	return a.Id == b.Id && token.Equal(a.Token, b.Token) && nodeEqual(a.Value, b.Value)
}

func printEqual(a, b PrintStatement) bool {
	return a.Format == b.Format && nodeEqual(a.Param, b.Param)
}

func assertEqual(a, b AssertStatement) bool {
	return token.Equal(a.Token, b.Token) && nodeEqual(a.Condition, b.Condition) && a.Message == b.Message
}

// A struct's fields are positional, so their order is part of the shape and not a detail of
// how the declaration was written.
func structDeclarationEqual(a, b StructDeclaration) bool {
	return a.Name == b.Name && slices.Equal(a.Fields, b.Fields)
}

// An import is the module it names and the name it is reached by. The same module under two
// aliases is two declarations, and so is two modules under one alias.
func useDeclarationEqual(a, b UseDeclaration) bool {
	return a.Specifier == b.Specifier && a.Alias == b.Alias
}

func structLiteralEqual(a, b StructLiteral) bool {
	return a.Name == b.Name && nodesEqual(a.Values, b.Values)
}

func fieldEqual(a, b FieldExpression) bool {
	return a.Index == b.Index && nodeEqual(a.Expression, b.Expression)
}

func shapedEqual(a, b ShapedExpression) bool {
	return a.Struct == b.Struct && nodeEqual(a.Expression, b.Expression)
}

func calleeEqual(a, b CalleeLiteral) bool {
	if !identifierEqual(a.Id, b.Id) || len(a.Params) != len(b.Params) {
		return false
	}
	for i := range a.Params {
		if !nodeEqual(a.Params[i].Expression, b.Params[i].Expression) {
			return false
		}
	}
	return true
}

func tapeEqual(a, b TapeBracketExpression) bool {
	return nodesEqual(a.Items, b.Items)
}

func feedEqual(a, b FeedExpression) bool {
	return numberEqual(a.Nth, b.Nth)
}

func nodesEqual(a, b []Node) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !nodeEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}
