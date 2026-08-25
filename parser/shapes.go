package parser

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
	"github.com/guiferpa/aurora/wire/token"
)

// What `shape` and `as` declare, and everything it is: a way of naming the fields of a run
// of tapes so the compiler can turn a name into an index, point at a mistake where it was
// written, and tell the language server what is there.
//
// None of it outlives the compilation. A shape value is a run of tapes and nothing else —
// the same run a reel of the same length is — so the tables here are read while parsing and
// dropped, and the tree carries indexes rather than names.

// Declarations are what those two keywords leave behind while a file is compiled: the fields
// each shape declared, and the shape a name is read as.
//
// They were called directives, which is what a language usually calls something outside its
// own grammar — a pragma, an annotation. These are inside it: `shape` and `as` are keywords,
// with a token, a parse rule and errors of their own. What sets them apart is not where they
// live but what they leave: a declaration, and no work to do.
//
// They belong to a source file rather than to a parse, which is what the REPL needs — there
// a file is typed one line at a time, with a fresh parser for each, and a shape declared on
// one line has to still be there on the next.
type Declarations struct {
	Shapes map[string][]string // shape name -> its fields, in order
	Reads  map[string]string   // identifier name -> the shape it is read as
	// Modules is what `use` declared: alias -> specifier. It belongs to the file that wrote
	// it and to no other, which is the whole point of the alias being mandatory.
	Modules map[string]string
	// Returns is the name of a scope -> the shape calling it returns. It is not the shape of
	// the name itself — a deferred scope is an index —
	// but the shape of what comes back from it.
	Returns map[string]string
}

func NewDeclarations() *Declarations {
	return &Declarations{
		Shapes:  make(map[string][]string),
		Reads:   make(map[string]string),
		Modules: make(map[string]string),
		Returns: make(map[string]string),
	}
}

// Import writes down what a module returns, under names nobody can type.
//
// What a scope returns crosses a module as the shape and its fields together, and both are written the
// way an identifier of that module is: with the module in front. So two modules returning
// an Env each are two shapes, and neither can be confused with an Env declared here.
func (d *Declarations) Import(specifier string, offer ast.Offer) {
	for _, shape := range offer.Shapes {
		d.Shapes[module.Qualify(module.ID(specifier), shape.Name)] = shape.Fields
	}
	for _, returns := range offer.Returns {
		// A scope may return a shape of a third module, which this one never declared, so the
		// fields come with it rather than being looked up.
		shape := module.Qualify(module.ID(specifier), returns.Shape)
		d.Shapes[shape] = returns.Fields
		d.Returns[module.Qualify(module.ID(specifier), returns.Scope)] = shape
	}
}

// capitalized says whether a name is written the way a shape's name has to be.
//
// The rule exists because a shape's name is the one name that is not a value. Everything else
// written in a file is something to load, and `Point{1, 2}` next to `point(1, 2)` is the
// difference between building a run of tapes and feeding a scope — a capital says which,
// before the braces do.
func capitalized(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(first)
}

// ParseShape reads `shape Point { x, y };`.
func (p *pr) ParseShape() (ast.Node, error) {
	tok, err := p.EatToken(token.SHAPE)
	if err != nil {
		return nil, err
	}

	name, err := p.EatToken(token.ID)
	if err != nil {
		return nil, err
	}
	shapeName := string(name.GetMatch())
	if !capitalized(shapeName) {
		return nil, token.NewError(name, "shape %s must start with a capital letter at line %d and column %d",
			shapeName, name.GetLine(), name.GetColumn())
	}
	if _, declared := p.declarations.Shapes[shapeName]; declared {
		return nil, token.NewError(name, "shape %s is already declared at line %d and column %d",
			shapeName, name.GetLine(), name.GetColumn())
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
			return nil, token.NewError(field, "shape %s already has a field named %s at line %d and column %d",
				shapeName, fieldName, field.GetLine(), field.GetColumn())
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
	// A shape of no fields would be a value of no tapes, which is not a value at all.
	if len(fields) == 0 {
		return nil, token.NewError(tok, "shape %s has no fields at line %d and column %d",
			shapeName, tok.GetLine(), tok.GetColumn())
	}

	p.declarations.Shapes[shapeName] = fields
	return ast.ShapeDeclaration{Name: shapeName, Fields: fields, Token: tok}, nil
}

// ParseShapeLiteral reads `Point{10, 20}`: one value per field, in the declared order.
//
// The braces are what tell a construction from applying values to a scope, which parentheses
// could not do on their own — `Point{1, 2}` and `greet(1, 2)` are the same shape. It is still
// only a construction when the name was declared, because `if flag { … }` also puts a brace
// after a name; declaring a shape called `flag` and testing on it is the one ambiguity left,
// and it is the same one Go has.
func (p *pr) ParseShapeLiteral(id ast.IdentifierLiteral) (ast.Node, error) {
	return p.parseShapeValue(typed(id), typed(id), id.Token)
}

// parseShapeValue reads `{ ... }` for a shape already known, whichever file declared it:
// shape is what it is called in the tables, and named is what whoever wrote the line typed.
func (p *pr) parseShapeValue(shape, named string, at token.Token) (ast.Node, error) {
	fields := p.declarations.Shapes[shape]

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

	// Pointing at a miscount is the whole reason the declaration exists, so it is an error
	// rather than a run padded with the neutral value.
	if len(values) != len(fields) {
		return nil, token.NewError(closing, "shape %s has %d fields (%s) but got %d at line %d and column %d",
			named, len(fields), strings.Join(fields, ", "), len(values), closing.GetLine(), closing.GetColumn())
	}

	return ast.ShapeLiteral{Name: shape, Values: values, Token: at}, nil
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

	// The head decides which of the two things a dot is. An alias is a name in this file's
	// scope and cannot also be a value, so there is nothing to be ambiguous about.
	if head, ok := expr.(ast.IdentifierLiteral); ok {
		if specifier, isModule := p.declarations.Modules[typed(head)]; isModule {
			return p.parseMember(specifier, field, name)
		}
	}

	shape := p.shapeOf(expr)
	if shape == "" {
		return nil, token.NewError(name, "cannot read field %s at line %d and column %d: nothing says which shape this value is, name it with 'as'",
			field, name.GetLine(), name.GetColumn())
	}

	fields := p.declarations.Shapes[shape]
	index := slices.Index(fields, field)
	if index < 0 {
		return nil, token.NewError(name, "shape %s has no field named %s at line %d and column %d (it has %s)",
			shape, field, name.GetLine(), name.GetColumn(), strings.Join(fields, ", "))
	}

	return ast.FieldExpression{
		Expression: expr,
		Index:      uint64(index),
		Fields:     uint64(len(fields)),
		Field:      field,
		Token:      name,
	}, nil
}

// parseShape reads `as Point`. It claims a shape rather than checking one: a value is a run
// of bytes and there is nothing in it to check against.
func (p *pr) parseShape(expr ast.Node) (ast.Node, error) {
	tok, err := p.EatToken(token.AS)
	if err != nil {
		return nil, err
	}
	shape, _, err := p.parseShapeName()
	if err != nil {
		return nil, err
	}

	return ast.ShapedExpression{Expression: expr, Shape: shape, Token: tok}, nil
}

// parseShapeName reads the name of a shape where one is expected: `Point`, or `m.Point` for
// one another module declared.
//
// The qualified form is read the same way a qualified value is, and gives the name the
// tables know it by — which nobody can type, so a Point of this file and a Point of another
// are two shapes and never one.
func (p *pr) parseShapeName() (string, token.Token, error) {
	name, err := p.EatToken(token.ID)
	if err != nil {
		return "", nil, err
	}
	shape := string(name.GetMatch())

	if specifier, isModule := p.declarations.Modules[shape]; isModule {
		if _, err := p.EatToken(token.DOT); err != nil {
			return "", nil, err
		}
		named, err := p.EatToken(token.ID)
		if err != nil {
			return "", nil, err
		}
		symbol := string(named.GetMatch())
		shape = module.Qualify(module.ID(specifier), symbol)
		if _, declared := p.declarations.Shapes[shape]; !declared {
			return "", nil, token.NewError(named, "module %s has no shape named %s at line %d and column %d",
				specifier, symbol, named.GetLine(), named.GetColumn())
		}
		return shape, named, nil
	}

	if _, declared := p.declarations.Shapes[shape]; !declared {
		return "", nil, token.NewError(name, "%s is not a declared shape at line %d and column %d",
			shape, name.GetLine(), name.GetColumn())
	}
	return shape, name, nil
}

// shapeOf says which shape an expression returns, or empty when nothing said. A field is
// one tape wide, so reading a field of a field is never known.
//
// What it sees through is what makes a declaration worth checking: a block returns its last
// expression, and an if returns whatever the arm that runs returns — so both arms have to agree,
// and an if with no else agrees with nothing.
func (p *pr) shapeOf(node ast.Node) string {
	switch n := node.(type) {
	case ast.ShapeLiteral:
		return n.Name
	case ast.ShapedExpression:
		return n.Shape
	case ast.IdentifierLiteral:
		return p.declarations.Reads[n.Value]
	case ast.BlockExpression:
		if n.Returns != "" {
			return n.Returns
		}
		return p.shapeOfLast(n.Body)
	case ast.CalleeLiteral:
		// Not the shape of the name, which is a deferred scope, but of what calling it
		// returns — which is only known because the scope declared it.
		return p.declarations.Returns[n.Id.Value]
	case ast.IfExpression:
		if n.Else == nil {
			return ""
		}
		shape := p.shapeOfLast(n.Body)
		if shape != "" && shape == p.shapeOfLast(n.Else.Body) {
			return shape
		}
	}
	return ""
}

// shapeOfLast says which shape the expression a body ends with returns, which is what the body returns.
func (p *pr) shapeOfLast(body []ast.Node) string {
	if len(body) == 0 {
		return ""
	}
	return p.shapeOf(body[len(body)-1])
}

// What `returns` declares, and how it is kept.
//
// `as` is a claim: the compiler believes it and has nothing to check it against, which is why
// a wrong one reads the wrong tape and the program carries on. `returns` is the other end —
// the block says what it returns, and this refuses the block that does not.
//
// The check is worth what shapeOf can see through, which is why that is the part that grew.

// returns is what the top-level scopes of this file return, with the fields of each — the only
// thing about a shape that leaves the file that declared it.
//
// It is a join of two tables the parse already filled: what each scope returns, and what each
// shape is made of. Whoever imports this file needs both halves, since a name is worth nothing
// without the fields it stands for.
func (p *pr) returns(nodes []ast.Node) []ast.Returns {
	found := make([]ast.Returns, 0)
	for _, node := range nodes {
		binding, ok := node.(ast.IdentLiteral)
		if !ok {
			continue
		}
		shape, known := p.declarations.Returns[binding.Id]
		if !known {
			continue
		}
		found = append(found, ast.Returns{
			Scope:  p.bare(binding.Id),
			Shape:  shape,
			Fields: p.declarations.Shapes[shape],
		})
	}
	return found
}

// shapes is every shape this file declared, with what each is made of.
//
// All of them cross, and not only the ones a scope returns: a file that imports this one may
// want to build one, or to name one with `as`, and neither goes through what a scope returns.
func (p *pr) shapes(nodes []ast.Node) []ast.Shape {
	found := make([]ast.Shape, 0)
	for _, node := range nodes {
		declaration, ok := node.(ast.ShapeDeclaration)
		if !ok {
			continue
		}
		found = append(found, ast.Shape{Name: declaration.Name, Fields: declaration.Fields})
	}
	return found
}

// bare is a name of this file without the module in front of it, which is how the file that
// wrote it reads it and how whoever imports it asks for it.
func (p *pr) bare(name string) string {
	return module.Module{ID: module.ID(p.module)}.Symbol(name)
}

// parseReturns reads `returns Person` after a block, and checks what the block ends with.
func (p *pr) parseReturns(body []ast.Node, closing token.Token) (string, error) {
	if p.GetLookahead() == nil || p.GetLookahead().GetTag().Id != token.RETURNS {
		return "", nil
	}
	if _, err := p.EatToken(token.RETURNS); err != nil {
		return "", err
	}
	declared, _, err := p.parseShapeName()
	if err != nil {
		return "", err
	}
	if err := p.returnsWhatItDeclared(body, declared, "", closing); err != nil {
		return "", err
	}
	return declared, nil
}

// returnsWhatItDeclared refuses a body that ends with something other than what was declared.
//
// A `where` names the arm being read, so a declaration broken inside a branch says which one; the
// block itself has no name and simply ends.
func (p *pr) returnsWhatItDeclared(body []ast.Node, declared, where string, at token.Token) error {
	if len(body) == 0 {
		return brokenDeclaration(at, declared, where, "nothing")
	}

	last := body[len(body)-1]
	// An if is looked through rather than at: it returns whatever the arm that runs
	// returns, so every arm has to keep the declaration. A branch is nested ifs by the time
	// anything reads the tree, so this covers it too.
	if answer, ok := last.(ast.IfExpression); ok {
		if answer.Else == nil {
			return token.NewError(at, "this block returns %s and its if has no else at line %d and column %d: one path returns nothing",
				declared, at.GetLine(), at.GetColumn())
		}
		if err := p.returnsWhatItDeclared(answer.Body, declared, "the if", at); err != nil {
			return err
		}
		return p.returnsWhatItDeclared(answer.Else.Body, declared, "the else", at)
	}

	if p.shapeOf(last) == declared {
		return nil
	}
	return brokenDeclaration(at, declared, where, describeReturn(last))
}

// brokenDeclaration says what was declared and what is there instead.
func brokenDeclaration(at token.Token, declared, where, returned string) error {
	if where == "" {
		return token.NewError(at, "this block returns %s and ends with %s at line %d and column %d",
			declared, returned, at.GetLine(), at.GetColumn())
	}
	return token.NewError(at, "this block returns %s and %s returns %s at line %d and column %d",
		declared, where, returned, at.GetLine(), at.GetColumn())
}

// describeReturn names what a node returns, for the message of a declaration that was not
// kept. It is the reader's words rather than the tree's: somebody reading the error is looking
// at what they wrote, not at a node type.
func describeReturn(node ast.Node) string {
	switch n := node.(type) {
	case ast.NumberLiteral:
		return "a number"
	case ast.TextLiteral:
		return "text"
	case ast.BooleanLiteral:
		return "a boolean"
	case ast.ShapeLiteral:
		return "a " + n.Name
	case ast.ShapedExpression:
		return "a " + n.Shape
	case ast.IdentifierLiteral:
		return "the name " + n.Value
	case ast.CalleeLiteral:
		return "a call to " + n.Id.Value
	case ast.DeferExpression:
		return "a deferred scope"
	case ast.TapeBracketExpression:
		return "a tape"
	case ast.BinaryExpression:
		return "arithmetic"
	case ast.RelativeExpression, ast.BooleanExpression:
		return "a comparison"
	}
	return "something no shape was named for"
}
