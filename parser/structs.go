package parser

import (
	"slices"
	"strings"

	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/token"
)

// The struct directives, and everything they are: a way of naming the fields of a run of
// tapes so the compiler can turn a name into an index, point at a mistake where it was
// written, and tell the language server what is there.
//
// None of it outlives the compilation. A struct value is a run of tapes and nothing else —
// the same run a reel of the same length is — so the tables here are read while parsing and
// dropped, and the tree carries indexes rather than names.

// Directives are what the struct directives leave behind while a file is compiled: the
// fields each struct declared, and the struct a name is read as.
//
// They belong to a source file rather than to a parse, which is what the REPL needs — there
// a file is typed one line at a time, with a fresh parser for each, and a struct declared on
// one line has to still be there on the next.
type Directives struct {
	Structs map[string][]string // struct name -> its fields, in order
	Shapes  map[string]string   // identifier name -> the struct it is read as
}

func NewDirectives() *Directives {
	return &Directives{
		Structs: make(map[string][]string),
		Shapes:  make(map[string]string),
	}
}

// ParseStruct reads `struct Point { x, y };`.
func (p *pr) ParseStruct() (ast.Node, error) {
	tok, err := p.EatToken(token.STRUCT)
	if err != nil {
		return nil, err
	}

	name, err := p.EatToken(token.ID)
	if err != nil {
		return nil, err
	}
	structName := string(name.GetMatch())
	if _, declared := p.directives.Structs[structName]; declared {
		return nil, token.NewError(name, "struct %s is already declared at line %d and column %d",
			structName, name.GetLine(), name.GetColumn())
	}

	if _, err := p.EatToken(token.O_CUR_BRK); err != nil {
		return nil, err
	}

	fields := make([]string, 0)
	for p.GetLookahead() != nil && p.GetLookahead().GetTag().Id != token.C_CUR_BRK {
		field, err := p.EatToken(token.ID)
		if err != nil {
			return nil, err
		}
		fieldName := string(field.GetMatch())
		if slices.Contains(fields, fieldName) {
			return nil, token.NewError(field, "struct %s already has a field named %s at line %d and column %d",
				structName, fieldName, field.GetLine(), field.GetColumn())
		}
		fields = append(fields, fieldName)

		if p.GetLookahead().GetTag().Id == token.C_CUR_BRK {
			break
		}
		if _, err := p.EatToken(token.COMMA); err != nil {
			return nil, err
		}
	}

	if _, err := p.EatToken(token.C_CUR_BRK); err != nil {
		return nil, err
	}
	// A struct of no fields would be a value of no tapes, which is not a value at all.
	if len(fields) == 0 {
		return nil, token.NewError(tok, "struct %s has no fields at line %d and column %d",
			structName, tok.GetLine(), tok.GetColumn())
	}

	p.directives.Structs[structName] = fields
	return ast.StructDeclaration{Name: structName, Fields: fields, Token: tok}, nil
}

// ParseStructLiteral reads `Point{10, 20}`: one value per field, in the declared order.
//
// The braces are what tell a construction from applying values to a scope, which parentheses
// could not do on their own — `Point{1, 2}` and `greet(1, 2)` are the same shape. It is still
// only a construction when the name was declared, because `if flag { … }` also puts a brace
// after a name; declaring a struct called `flag` and testing on it is the one ambiguity left,
// and it is the same one Go has.
func (p *pr) ParseStructLiteral(id ast.IdentifierLiteral) (ast.Node, error) {
	fields := p.directives.Structs[id.Value]

	if _, err := p.EatToken(token.O_CUR_BRK); err != nil {
		return nil, err
	}
	values := make([]ast.Node, 0, len(fields))
	for p.GetLookahead() != nil && p.GetLookahead().GetTag().Id != token.C_CUR_BRK {
		expr, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		values = append(values, expr)

		if p.GetLookahead().GetTag().Id == token.C_CUR_BRK {
			break
		}
		if _, err := p.EatToken(token.COMMA); err != nil {
			return nil, err
		}
	}
	closing, err := p.EatToken(token.C_CUR_BRK)
	if err != nil {
		return nil, err
	}

	// Pointing at a miscount is the whole reason the directive exists, so it is an error
	// rather than a run padded with the neutral value.
	if len(values) != len(fields) {
		return nil, token.NewError(closing, "struct %s has %d fields (%s) but got %d at line %d and column %d",
			id.Value, len(fields), strings.Join(fields, ", "), len(values), closing.GetLine(), closing.GetColumn())
	}

	return ast.StructLiteral{Name: id.Value, Values: values, Token: id.Token}, nil
}

// parsePostfix applies what binds tightest of all: reading a field, and naming the shape a
// value is read with. Both are left to right, so `feed(0) as Point.x` reads the field of
// the shaped value.
func (p *pr) parsePostfix(expr ast.Node) (ast.Node, error) {
	for {
		lookahead := p.GetLookahead()
		if lookahead == nil {
			return expr, nil
		}

		var err error
		switch lookahead.GetTag().Id {
		case token.DOT:
			expr, err = p.parseField(expr)
		case token.AS:
			expr, err = p.parseShape(expr)
		default:
			return expr, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

// parseField resolves `.x` to the index x was declared at. The name does not survive into
// the tree as anything the emitter reads.
func (p *pr) parseField(expr ast.Node) (ast.Node, error) {
	if _, err := p.EatToken(token.DOT); err != nil {
		return nil, err
	}
	name, err := p.EatToken(token.ID)
	if err != nil {
		return nil, err
	}
	field := string(name.GetMatch())

	shape := p.shapeOf(expr)
	if shape == "" {
		return nil, token.NewError(name, "cannot read field %s at line %d and column %d: nothing says which struct this value is, name it with 'as'",
			field, name.GetLine(), name.GetColumn())
	}

	fields := p.directives.Structs[shape]
	index := slices.Index(fields, field)
	if index < 0 {
		return nil, token.NewError(name, "struct %s has no field named %s at line %d and column %d (it has %s)",
			shape, field, name.GetLine(), name.GetColumn(), strings.Join(fields, ", "))
	}

	return ast.FieldExpression{Expression: expr, Index: uint64(index), Field: field, Token: name}, nil
}

// parseShape reads `as Point`. It claims a shape rather than checking one: a value is a run
// of bytes and there is nothing in it to check against.
func (p *pr) parseShape(expr ast.Node) (ast.Node, error) {
	tok, err := p.EatToken(token.AS)
	if err != nil {
		return nil, err
	}
	name, err := p.EatToken(token.ID)
	if err != nil {
		return nil, err
	}
	structName := string(name.GetMatch())
	if _, declared := p.directives.Structs[structName]; !declared {
		return nil, token.NewError(name, "%s is not a declared struct at line %d and column %d",
			structName, name.GetLine(), name.GetColumn())
	}

	return ast.ShapedExpression{Expression: expr, Struct: structName, Token: tok}, nil
}

// shapeOf answers which struct a value is read as, or empty when nothing said. A field is
// one tape wide, so reading a field of a field is never known.
func (p *pr) shapeOf(node ast.Node) string {
	switch n := node.(type) {
	case ast.StructLiteral:
		return n.Name
	case ast.ShapedExpression:
		return n.Struct
	case ast.IdentifierLiteral:
		return p.directives.Shapes[n.Value]
	}
	return ""
}
