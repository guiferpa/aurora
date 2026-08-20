package parser

import (
	"slices"
	"strings"

	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
	"github.com/guiferpa/aurora/wire/token"
)

// What `struct` and `as` declare, and everything it is: a way of naming the fields of a run
// of tapes so the compiler can turn a name into an index, point at a mistake where it was
// written, and tell the language server what is there.
//
// None of it outlives the compilation. A struct value is a run of tapes and nothing else —
// the same run a reel of the same length is — so the tables here are read while parsing and
// dropped, and the tree carries indexes rather than names.

// Declarations are what those two keywords leave behind while a file is compiled: the fields
// each struct declared, and the struct a name is read as.
//
// They were called directives, which is what a language usually calls something outside its
// own grammar — a pragma, an annotation. These are inside it: `struct` and `as` are keywords,
// with a token, a parse rule and errors of their own. What sets them apart is not where they
// live but what they leave: a declaration, and no work to do.
//
// They belong to a source file rather than to a parse, which is what the REPL needs — there
// a file is typed one line at a time, with a fresh parser for each, and a struct declared on
// one line has to still be there on the next.
type Declarations struct {
	Structs map[string][]string // struct name -> its fields, in order
	Shapes  map[string]string   // identifier name -> the struct it is read as
	// Modules is what `use` declared: alias -> specifier. It belongs to the file that wrote
	// it and to no other, which is the whole point of the alias being mandatory.
	Modules map[string]string
	// Returns is what `returns` promised: the name of a scope -> the struct that calling it
	// answers with. It is not the shape of the name itself — a deferred scope is an index —
	// but the shape of what comes back from it.
	Returns map[string]string
}

func NewDeclarations() *Declarations {
	return &Declarations{
		Structs: make(map[string][]string),
		Shapes:  make(map[string]string),
		Modules: make(map[string]string),
		Returns: make(map[string]string),
	}
}

// Import writes down what a module answers with, under names nobody can type.
//
// A promise crossing a module is the shape and its fields together, and both are written the
// way an identifier of that module is: with the module in front. So two modules answering with
// an Env each are two shapes, and neither can be confused with an Env declared here.
func (d *Declarations) Import(specifier string, offer ast.Offer) {
	for _, shape := range offer.Shapes {
		d.Structs[module.Qualify(module.ID(specifier), shape.Name)] = shape.Fields
	}
	for _, promise := range offer.Promises {
		// A promise may name a shape of a third module, which this one never declared, so the
		// fields come with it rather than being looked up.
		shape := module.Qualify(module.ID(specifier), promise.Struct)
		d.Structs[shape] = promise.Fields
		d.Returns[module.Qualify(module.ID(specifier), promise.Scope)] = shape
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
	if _, declared := p.declarations.Structs[structName]; declared {
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

	p.declarations.Structs[structName] = fields
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
	return p.parseStructValue(typed(id), typed(id), id.Token)
}

// parseStructValue reads `{ ... }` for a struct already known, whichever file declared it:
// shape is what it is called in the tables, and named is what whoever wrote the line typed.
func (p *pr) parseStructValue(shape, named string, at token.Token) (ast.Node, error) {
	fields := p.declarations.Structs[shape]

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
		return nil, token.NewError(closing, "struct %s has %d fields (%s) but got %d at line %d and column %d",
			named, len(fields), strings.Join(fields, ", "), len(values), closing.GetLine(), closing.GetColumn())
	}

	return ast.StructLiteral{Name: shape, Values: values, Token: at}, nil
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
		return nil, token.NewError(name, "cannot read field %s at line %d and column %d: nothing says which struct this value is, name it with 'as'",
			field, name.GetLine(), name.GetColumn())
	}

	fields := p.declarations.Structs[shape]
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
	shape, _, err := p.parseShapeName()
	if err != nil {
		return nil, err
	}

	return ast.ShapedExpression{Expression: expr, Struct: shape, Token: tok}, nil
}

// parseShapeName reads the name of a struct where one is expected: `Point`, or `m.Point` for
// one another module declared.
//
// The qualified form is read the same way a qualified value is, and answers with the name the
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
		if _, declared := p.declarations.Structs[shape]; !declared {
			return "", nil, token.NewError(named, "module %s has no struct named %s at line %d and column %d",
				specifier, symbol, named.GetLine(), named.GetColumn())
		}
		return shape, named, nil
	}

	if _, declared := p.declarations.Structs[shape]; !declared {
		return "", nil, token.NewError(name, "%s is not a declared struct at line %d and column %d",
			shape, name.GetLine(), name.GetColumn())
	}
	return shape, name, nil
}

// shapeOf answers which struct a value is read as, or empty when nothing said. A field is
// one tape wide, so reading a field of a field is never known.
//
// What it sees through is what a promise is worth: a block answers with its last expression,
// and an if answers with whatever the arm that runs answers with — so both arms have to agree,
// and an if with no else agrees with nothing.
func (p *pr) shapeOf(node ast.Node) string {
	switch n := node.(type) {
	case ast.StructLiteral:
		return n.Name
	case ast.ShapedExpression:
		return n.Struct
	case ast.IdentifierLiteral:
		return p.declarations.Shapes[n.Value]
	case ast.BlockExpression:
		if n.Returns != "" {
			return n.Returns
		}
		return p.shapeOfLast(n.Body)
	case ast.CalleeLiteral:
		// Not the shape of the name, which is a deferred scope, but of what calling it
		// answers with — which is only known because the scope promised.
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

// shapeOfLast answers for the expression a body ends with, which is what the body answers with.
func (p *pr) shapeOfLast(body []ast.Node) string {
	if len(body) == 0 {
		return ""
	}
	return p.shapeOf(body[len(body)-1])
}

// What `returns` promises, and how the promise is kept.
//
// `as` is a claim: the compiler believes it and has nothing to check it against, which is why
// a wrong one reads the wrong tape and the program carries on. `returns` is the other end —
// the block says what it answers with, and this refuses the block that does not.
//
// The promise is worth what shapeOf can see through, which is why that is the part that grew.

// promises is what the top-level scopes of this file said they answer with, with the fields
// of each — the only thing about a struct that leaves the file that declared it.
//
// It is a join of two tables the parse already filled: which scope promised what, and what
// each struct is made of. Whoever imports this file needs both halves, since a name is worth
// nothing without the fields it stands for.
func (p *pr) promises(nodes []ast.Node) []ast.Promise {
	found := make([]ast.Promise, 0)
	for _, node := range nodes {
		binding, ok := node.(ast.IdentLiteral)
		if !ok {
			continue
		}
		promised, made := p.declarations.Returns[binding.Id]
		if !made {
			continue
		}
		found = append(found, ast.Promise{
			Scope:  p.bare(binding.Id),
			Struct: promised,
			Fields: p.declarations.Structs[promised],
		})
	}
	return found
}

// shapes is every struct this file declared, with what each is made of.
//
// All of them cross, and not only the ones a promise names: a file that imports this one may
// want to build one, or to name one with `as`, and neither goes through a promise.
func (p *pr) shapes(nodes []ast.Node) []ast.Shape {
	found := make([]ast.Shape, 0)
	for _, node := range nodes {
		declaration, ok := node.(ast.StructDeclaration)
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
	promised, _, err := p.parseShapeName()
	if err != nil {
		return "", err
	}
	if err := p.answersWith(body, promised, "", closing); err != nil {
		return "", err
	}
	return promised, nil
}

// answersWith refuses a body that ends with something other than what was promised.
//
// A `where` names the arm being read, so a promise broken inside a branch says which one; the
// block itself has no name and simply ends.
func (p *pr) answersWith(body []ast.Node, promised, where string, at token.Token) error {
	if len(body) == 0 {
		return brokenPromise(at, promised, where, "nothing")
	}

	last := body[len(body)-1]
	// An if is looked through rather than at: it answers with whatever the arm that runs
	// answers with, so every arm has to keep the promise. A branch is nested ifs by the time
	// anything reads the tree, so this covers it too.
	if answer, ok := last.(ast.IfExpression); ok {
		if answer.Else == nil {
			return token.NewError(at, "this block answers with %s and its if has no else at line %d and column %d: one path answers with nothing",
				promised, at.GetLine(), at.GetColumn())
		}
		if err := p.answersWith(answer.Body, promised, "the if", at); err != nil {
			return err
		}
		return p.answersWith(answer.Else.Body, promised, "the else", at)
	}

	if p.shapeOf(last) == promised {
		return nil
	}
	return brokenPromise(at, promised, where, describeAnswer(last))
}

// brokenPromise says what was promised and what is there instead.
func brokenPromise(at token.Token, promised, where, answer string) error {
	if where == "" {
		return token.NewError(at, "this block answers with %s and ends with %s at line %d and column %d",
			promised, answer, at.GetLine(), at.GetColumn())
	}
	return token.NewError(at, "this block answers with %s and %s answers with %s at line %d and column %d",
		promised, where, answer, at.GetLine(), at.GetColumn())
}

// describeAnswer names what a node answers with, for the message of a promise that was not
// kept. It is the reader's words rather than the tree's: somebody reading the error is looking
// at what they wrote, not at a node type.
func describeAnswer(node ast.Node) string {
	switch n := node.(type) {
	case ast.NumberLiteral:
		return "a number"
	case ast.TextLiteral:
		return "text"
	case ast.BooleanLiteral:
		return "a boolean"
	case ast.StructLiteral:
		return "a " + n.Name
	case ast.ShapedExpression:
		return "a " + n.Struct
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
