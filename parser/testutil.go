// Package parser: test helpers for AST comparison (same package so tests can use them without import cycle).

package parser

import (
	"bytes"
	"reflect"

	"github.com/guiferpa/aurora/lexer"
)

// TokenEqual compares two lexer.Token by value (GetMatch, GetTag).
func TokenEqual(a, b lexer.Token) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return bytes.Equal(a.GetMatch(), b.GetMatch()) && a.GetTag() == b.GetTag()
}

// ModuleEqual compares two Module ASTs by structure and token value (ignores pointer identity).
func ModuleEqual(got, want Module) bool {
	if got.Name != want.Name || len(got.Statements) != len(want.Statements) {
		return false
	}
	for i := range got.Statements {
		if !nodeEqual(got.Statements[i], want.Statements[i]) {
			return false
		}
	}
	return true
}

func nodeEqual(a, b Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch va := a.(type) {
	case Statement:
		vb, ok := b.(Statement)
		if !ok {
			return false
		}
		return nodeEqual(va.Node, vb.Node)
	case NothingLiteral:
		vb, ok := b.(NothingLiteral)
		if !ok {
			return false
		}
		return TokenEqual(va.Token, vb.Token)
	case NumberLiteral:
		vb, ok := b.(NumberLiteral)
		if !ok {
			return false
		}
		return va.Value == vb.Value && TokenEqual(va.Token, vb.Token)
	case BooleanLiteral:
		vb, ok := b.(BooleanLiteral)
		if !ok {
			return false
		}
		return bytes.Equal(va.Value, vb.Value) && TokenEqual(va.Token, vb.Token)
	case BinaryExpression:
		vb, ok := b.(BinaryExpression)
		if !ok {
			return false
		}
		return nodeEqual(va.Left, vb.Left) && nodeEqual(va.Right, vb.Right) && opEqual(va.Operation, vb.Operation)
	case IfExpression:
		vb, ok := b.(IfExpression)
		if !ok {
			return false
		}
		if !nodeEqual(va.Test, vb.Test) || !nodesEqual(va.Body, vb.Body) {
			return false
		}
		if (va.Else == nil) != (vb.Else == nil) {
			return false
		}
		if va.Else != nil && !elseEqual(*va.Else, *vb.Else) {
			return false
		}
		return true
	case BlockExpression:
		vb, ok := b.(BlockExpression)
		if !ok {
			return false
		}
		return nodesEqual(va.Body, vb.Body)
	case IdentStatement:
		vb, ok := b.(IdentStatement)
		if !ok {
			return false
		}
		return va.Id == vb.Id && TokenEqual(va.Token, vb.Token) && nodeEqual(va.Expression, vb.Expression)
	case PrintStatement:
		vb, ok := b.(PrintStatement)
		if !ok {
			return false
		}
		return nodeEqual(va.Param, vb.Param)
	case EchoStatement:
		vb, ok := b.(EchoStatement)
		if !ok {
			return false
		}
		return nodeEqual(va.Param, vb.Param)
	case AssertStatement:
		vb, ok := b.(AssertStatement)
		if !ok {
			return false
		}
		return TokenEqual(va.Token, vb.Token) && nodeEqual(va.Condition, vb.Condition) && nodeEqual(va.Message, vb.Message)
	case UnaryExpression:
		vb, ok := b.(UnaryExpression)
		if !ok {
			return false
		}
		return nodeEqual(va.Expression, vb.Expression) && opEqual(va.Operation, vb.Operation)
	case DeferExpression:
		vb, ok := b.(DeferExpression)
		if !ok {
			return false
		}
		return blockEqual(va.Block, vb.Block)
	case CalleeLiteral:
		vb, ok := b.(CalleeLiteral)
		if !ok {
			return false
		}
		if !identifierEqual(va.Id, vb.Id) {
			return false
		}
		if len(va.Params) != len(vb.Params) {
			return false
		}
		for i := range va.Params {
			if !nodeEqual(va.Params[i].Expression, vb.Params[i].Expression) {
				return false
			}
		}
		return true
	case IdentifierLiteral:
		vb, ok := b.(IdentifierLiteral)
		if !ok {
			return false
		}
		return va.Value == vb.Value && TokenEqual(va.Token, vb.Token)
	case OperationLiteral:
		vb, ok := b.(OperationLiteral)
		if !ok {
			return false
		}
		return opEqual(va, vb)
	case TapeBracketExpression:
		vb, ok := b.(TapeBracketExpression)
		if !ok {
			return false
		}
		if len(va.Items) != len(vb.Items) {
			return false
		}
		for i := range va.Items {
			if !nodeEqual(va.Items[i], vb.Items[i]) {
				return false
			}
		}
		return true
	case ArgumentsExpression:
		vb, ok := b.(ArgumentsExpression)
		if !ok {
			return false
		}
		return va.Nth.Value == vb.Nth.Value && TokenEqual(va.Nth.Token, vb.Nth.Token)
	case RelativeExpression:
		vb, ok := b.(RelativeExpression)
		if !ok {
			return false
		}
		return nodeEqual(va.Left, vb.Left) && nodeEqual(va.Right, vb.Right) && opEqual(va.Operation, vb.Operation)
	case BooleanExpression:
		vb, ok := b.(BooleanExpression)
		if !ok {
			return false
		}
		return nodeEqual(va.Left, vb.Left) && nodeEqual(va.Right, vb.Right) && opEqual(va.Operation, vb.Operation)
	default:
		return reflect.DeepEqual(a, b)
	}
}

func opEqual(a, b OperationLiteral) bool {
	return a.Value == b.Value && TokenEqual(a.Token, b.Token)
}

func blockEqual(a, b BlockExpression) bool {
	return nodesEqual(a.Body, b.Body)
}

func elseEqual(a, b ElseExpression) bool {
	return nodesEqual(a.Body, b.Body)
}

func identifierEqual(a, b IdentifierLiteral) bool {
	return a.Value == b.Value && TokenEqual(a.Token, b.Token)
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
