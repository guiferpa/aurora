package ast

import (
	"github.com/guiferpa/aurora/wire/token"
)

// A Node is one node of the tree. The method is unexported and does nothing: it is a mark,
// and marking is all the interface ever did.
//
// Being unexported closes the set — a node can only be declared here. The emitter switches
// over every concrete type there is, so a node declared elsewhere would compile and then fall
// through that switch in silence.
//
// The mark used to be "Next() Node", implemented thirty-one times and called never. It could
// not have been the walk it read like: this tree is n-ary — seven types hold no child, eleven
// hold one, three hold two, three hold three (a binary expression included, since the operator
// is a node as well) and seven hold a list — and Next answered with a single child, so
// twenty-five of the thirty-one answered nil. The walk that does exist is a type switch, in
// emitter/warning.go, which is what asking a tree about its shape actually takes.
type Node interface {
	node()
}

// mark is what makes a type a node, embedded as the first field of every one of them.
//
// Embedding rather than writing the method thirty-one times: only this package can embed an
// unexported type, so the set is closed just the same, and a node type declares what it is on
// its first line instead of somewhere else in the file. It holds nothing and comes first, so
// it costs no memory.
type mark struct{}

func (mark) node() {}

type OperationLiteral struct {
	mark
	Value string      `json:"value"`
	Token token.Token `json:"-"`
}

type ParameterLiteral struct {
	mark
	Expression Node `json:"expression"`
}

type CalleeLiteral struct {
	mark
	Id     IdentifierLiteral  `json:"identifier"`
	Params []ParameterLiteral `json:"parameters"`
}

type IdentifierLiteral struct {
	mark
	Value string      `json:"value"`
	Token token.Token `json:"-"`
}

type BooleanLiteral struct {
	mark
	Value []byte      `json:"value"`
	Token token.Token `json:"-"`
}

type NumberLiteral struct {
	mark
	Value uint64      `json:"value"`
	Token token.Token `json:"-"`
}

// TextLiteral is `"text"`: the bytes of the text, packed into one tape.
//
// It is one more way of writing a tape, next to `1`, `0x2a`, `[1, 2]` and `true` — not a
// kind of value of its own.
//
// Text used to be a reel — one tape per character, so "Gui" was three tapes and 24 bytes at
// the default width. That was a way of holding more than a tape could, and it cost
// tape_size - 1 bytes of zero for every character. Text is now the bytes it is, right
// aligned like every other value, so "Gui" is three bytes and "a" is 97, the same tape the
// number 97 is.
type TextLiteral struct {
	mark
	Value []byte      `json:"value"`
	Token token.Token `json:"-"`
}

type UnaryExpression struct {
	mark
	Expression Node             `json:"expression"`
	Operation  OperationLiteral `json:"operation"`
}

type BinaryExpression struct {
	mark
	Left      Node             `json:"left"`
	Right     Node             `json:"right"`
	Operation OperationLiteral `json:"operation"`
}

type PrimaryExpression struct {
	mark
	Expression Node `json:"expression"`
}

type TapeExpression struct {
	mark
	Length uint64 `json:"length"`
}

type TapeBracketExpression struct {
	mark
	Items []Node      `json:"items"`
	Token token.Token `json:"-"`
}

type PullExpression struct {
	mark
	Target Node        `json:"target"`
	Item   Node        `json:"item"`
	Token  token.Token `json:"-"`
}

type HeadExpression struct {
	mark
	Expression Node        `json:"expression"`
	Length     uint64      `json:"length"`
	Token      token.Token `json:"-"`
}

type TailExpression struct {
	mark
	Expression Node        `json:"expression"`
	Length     uint64      `json:"length"`
	Token      token.Token `json:"-"`
}

type PushExpression struct {
	mark
	Target Node        `json:"target"`
	Item   Node        `json:"item"`
	Token  token.Token `json:"-"`
}

type RelativeExpression struct {
	mark
	Left      Node             `json:"left"`
	Right     Node             `json:"right"`
	Operation OperationLiteral `json:"operation"`
}

type BooleanExpression struct {
	mark
	Left      Node             `json:"left"`
	Right     Node             `json:"right"`
	Operation OperationLiteral `json:"operation"`
}

type BlockExpression struct {
	mark
	// Returns is the shape this block declared it returns, and empty when it declared
	// nothing. It is checked where it is written and never reaches an instruction: like the
	// declaration it names, it is read while compiling and dropped.
	Returns string `json:"returns,omitempty"`
	Body    []Node `json:"body"`
}

// DeferExpression is "defer { ... }". It produces a value that is a pointer to the scope
// (executable later via invocation, e.g. r(1, 2)). No signature or arity.
// Block is the body of the defer; it is a BlockExpression so the emitter can treat it
// as a normal scope (BeginScope + body + Return) without duplicating scope logic.
type DeferExpression struct {
	mark
	Block BlockExpression `json:"block"`
}

type IfExpression struct {
	mark
	Test  Node            `json:"test"`
	Body  []Node          `json:"body"`
	Else  *ElseExpression `json:"else"`
	Token token.Token     `json:"-"`
}

type ElseExpression struct {
	mark
	Body []Node `json:"body"`
}

// PrintFormat is how a print builtin reads the tape it is given.
type PrintFormat string

const (
	PrintBytes   PrintFormat = "bytes"   // printb
	PrintChars   PrintFormat = "chars"   // printc
	PrintDecimal PrintFormat = "decimal" // printd
)

// PrintStatement is one of the print builtins. They differ only in how the value is read,
// so they are one node carrying which reading was asked for.
type PrintStatement struct {
	mark
	Format PrintFormat `json:"format"`
	Param  Node        `json:"parameter"`
	Token  token.Token `json:"-"`
}

// FeedExpression is "feed(n)": it reads the nth value applied to the running scope.
type FeedExpression struct {
	mark
	Nth NumberLiteral `json:"nth"`
}

type IdentLiteral struct {
	mark
	Id    string      `json:"id"`
	Token token.Token `json:"-"`
	Value Node        `json:"value"`
}

// AssertStatement is `assert(condition, "message")`.
//
// The message is a literal held as text, not as a value: it is written for whoever reads
// the result of a test, the same way a shape's field names are written for whoever reads
// the source. Keeping it out of the values is also what lets it be longer than a tape,
// which a message usually is.
type AssertStatement struct {
	mark
	Condition Node        `json:"condition"`
	Message   string      `json:"message"`
	Token     token.Token `json:"-"`
}

// AST is the top-level node: Aurora is expression-only, so a parsed file is the sequence
// of expressions it holds. The unit of compilation is the file.
type AST struct {
	mark
	Filename string `json:"filename"`
	Nodes    []Node `json:"nodes"`
	// References is every qualified name the file used — `x.add` after `use a/b as x;`.
	//
	// It rides alongside the tree instead of in it because it is not shape: the nodes already
	// carry the name the instruction will hold, and this is the list of what has to be found
	// somewhere else. Only whoever holds the other modules can answer that, which is why the
	// parse hands it over instead of deciding.
	References []Reference `json:"references,omitempty"`
	// Returns is what this file's top-level scopes return, and Shapes is every shape it
	// declared. Both leave with the tree for whoever imports this file.
	Returns []Returns `json:"returns,omitempty"`
	Shapes  []Shape   `json:"shapes,omitempty"`
}

// An Offer is what a module hands whoever imports it, as far as shapes are concerned: the
// shapes it declares, and what its scopes return.
type Offer struct {
	Shapes  []Shape   `json:"shapes,omitempty"`
	Returns []Returns `json:"returns,omitempty"`
}

// A Shape is a shape a module declared, and what it is made of.
//
// The name does not cross on its own — it is written in another file as the module's, never
// as itself — and the fields cross with it, because an index is a position and a position
// needs the whole list.
type Shape struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
}

// Returns is what one exported scope returns, and what that shape is made of.
//
// It is the only thing about a shape that leaves the file that declared it. A shape's name is
// read while a file is parsed and dropped, so a module that returns one has to hand over the
// fields as well — otherwise whoever imports it has a run of tapes and no way to turn a name
// into an index.
//
// The name is plural because that is the word: a scope returns, and this is what it returns.
// It used to be called a promise, which came from another tradition and said nothing this
// language means — see "The words" in docs/contributing/architecture.md.
type Returns struct {
	// Scope is the name the scope is bound to, as the module's own file typed it.
	Scope  string   `json:"scope"`
	Shape  string   `json:"shape"`
	Fields []string `json:"fields"`
}

// A Reference is one qualified name, and where it was written.
type Reference struct {
	// Module is the module the name belongs to — the specifier, not the alias, because the
	// alias means nothing outside the file that declared it.
	Module string      `json:"module"`
	Symbol string      `json:"symbol"`
	Token  token.Token `json:"-"`
}

// ShapeDeclaration names the fields of a run of tapes: `shape Point { x, y };`.
//
// It declares a shape for whoever writes the source; it is not a value. The compiler reads it to
// turn a field name into an index, to report a mistake where it was written, and to feed
// the language server — and then it is gone. Nothing about it reaches the IR: the flow is
// static, the fields are positional and every one of them is exactly a tape wide.
type ShapeDeclaration struct {
	mark
	Name   string      `json:"name"`
	Fields []string    `json:"fields"`
	Token  token.Token `json:"-"`
}

// ShapeLiteral builds the run: `Point{10, 20}` is two tapes, one per field.
type ShapeLiteral struct {
	mark
	Name   string      `json:"name"`
	Values []Node      `json:"values"`
	Token  token.Token `json:"-"`
}

// FieldExpression reads one tape out of a run. The index is resolved while parsing, from
// the shape of the value, so nothing about the field's name survives here.
type FieldExpression struct {
	mark
	Expression Node   `json:"expression"`
	Index      uint64 `json:"index"`
	// Fields is how many tapes the run has, which the declaration the index came from also
	// says. Where a field sits inside a run cannot be worked out from the index alone: a run
	// is tapes laid end to end with nothing marking where it ends, so a target that keeps one
	// as a fixed-width value has to be told how long it is.
	Fields uint64      `json:"fields"`
	Field  string      `json:"field"` // kept for the language server, never emitted
	Token  token.Token `json:"-"`
}

// ShapedExpression is `expr as Point`: it says how to read a value the compiler cannot
// trace back to a construction. It has no effect of its own — the emitter emits what is
// inside it and drops the name.
type ShapedExpression struct {
	mark
	Expression Node        `json:"expression"`
	Shape      string      `json:"shape"`
	Token      token.Token `json:"-"`
}

// UseDeclaration brings a module in under an alias: `use a/b/c as x;`.
//
// It declares rather than binds. The alias names something only the compiler resolves — a
// module — and not a value, so nothing prints it, passes it or keeps it, exactly like the
// name a shape declares. That is also why it emits no work.
type UseDeclaration struct {
	mark
	// Specifier is the module's path from the source root, without the extension: "a/b/c"
	// reads a/b/c.ar. It is the module's identity, and it is the same text for every file
	// that imports it, which is what makes it the key of everything downstream.
	Specifier string `json:"specifier"`
	// Alias is how this file reaches what is inside, and it belongs to this file alone.
	Alias string      `json:"alias"`
	Token token.Token `json:"-"`
}
