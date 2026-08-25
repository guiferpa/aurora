package textdoc

import (
	"slices"
	"strconv"

	"github.com/guiferpa/aurora/wire/token"
)

// What the language server knows about the shape declarations, read straight from the tokens
// rather than from the tree.
//
// It has to be the tokens. A document being edited is broken most of the time — the moment
// someone types `p.` there is no field name yet and nothing parses — and completion is
// exactly what is wanted at that moment. The parser drops its own tables when the parse
// ends, and would have nothing to offer here anyway.
type shapeTable struct {
	fields map[string][]string // shape name -> its fields, in order
	reads  map[string]string   // identifier name -> the shape it is read as
	// returns is the name of a scope -> the shape calling it gives back. A name bound from
	// such a call is read as that shape without anybody claiming it.
	returns map[string]string
}

// newShapeTable is what nothing has been read into yet.
func newShapeTable() shapeTable {
	return shapeTable{
		fields:  make(map[string][]string),
		reads:   make(map[string]string),
		returns: make(map[string]string),
	}
}

// scan reads the declarations out of a token stream, on top of whatever is already known:
// `shape Point { x, y }` for the fields, and `ident p = Point{...}` or `... as Point` for
// what a name is read as.
//
// It reads on top rather than from nothing because a document is not the only thing that
// declares a shape. What its modules offer is written down first, and a name bound here from
// a call over there is only read when both halves are in the same table.
func (found shapeTable) scan(tokens []token.Token) shapeTable {
	for i := 0; i < len(tokens); i++ {
		switch tokens[i].GetTag().Id {
		case token.SHAPE:
			if name, fields, next := readDeclaration(tokens, i); name != "" {
				found.fields[name] = fields
				i = next
			}
		case token.IDENT:
			// `ident p = Point{`, `ident p = <anything> as Point` and a call to a scope that
			// promised all say what p is; `ident f = defer { ... } returns Point` says what
			// calling f will.
			name, shape, declared, called := readBinding(tokens, i)
			if declared != "" {
				found.returns[name] = declared
			}
			if shape == "" && called != "" {
				shape = found.returns[called]
			}
			if shape != "" {
				found.reads[name] = shape
			}
		}
	}

	return found
}

// readDeclaration reads `shape Name { a, b }` starting at the shape token.
func readDeclaration(tokens []token.Token, i int) (string, []string, int) {
	if i+2 >= len(tokens) || tokens[i+1].GetTag().Id != token.ID || tokens[i+2].GetTag().Id != token.O_CUR_BRK {
		return "", nil, i
	}

	name := string(tokens[i+1].GetMatch())
	fields := make([]string, 0)
	for j := i + 3; j < len(tokens); j++ {
		switch tokens[j].GetTag().Id {
		case token.ID:
			fields = append(fields, string(tokens[j].GetMatch()))
		case token.COMMA:
		case token.C_CUR_BRK:
			return name, fields, j
		default:
			return name, fields, j
		}
	}
	return name, fields, len(tokens) - 1
}

// readBinding reads what `ident name = ...` binds, up to the end of the statement: the shape
// it is read as, the shape calling it will answer with, and the scope it was bound from.
//
// Only what sits at the top of the value is read. A body between braces is another scope's
// business — the semicolons in it do not end this statement, and a construction inside it says
// what that scope answers with rather than what this name is.
func readBinding(tokens []token.Token, i int) (name, shape, promised, called string) {
	if i+2 >= len(tokens) || tokens[i+1].GetTag().Id != token.ID || tokens[i+2].GetTag().Id != token.ASSIGN {
		return "", "", "", ""
	}
	name = string(tokens[i+1].GetMatch())

	depth := 0
	for j := i + 3; j < len(tokens); j++ {
		switch tokens[j].GetTag().Id {
		case token.O_CUR_BRK:
			depth++
		case token.C_CUR_BRK:
			depth--
		case token.SEMICOLON:
			if depth <= 0 {
				return name, shape, promised, called
			}
		case token.RETURNS:
			if claimed := claimedAt(tokens, j+1); depth == 0 && claimed != "" {
				promised = claimed
			}
		case token.AS:
			// The nearest `as` wins, which is the one the value ends up with.
			if claimed := claimedAt(tokens, j+1); depth == 0 && claimed != "" {
				shape = claimed
			}
		case token.ID:
			if depth != 0 || j+1 >= len(tokens) {
				continue
			}
			switch tokens[j+1].GetTag().Id {
			case token.O_CUR_BRK:
				// A construction: the name of a shape in front of a brace.
				shape = nameAt(tokens, j)
			case token.O_PAREN:
				// A call: what it answers with is known when that scope promised.
				called = nameAt(tokens, j)
			}
		}
	}
	return name, shape, promised, called
}

// nameAt reads the name at this position the way the document wrote it: `Square`, or
// `g.Square` when it hangs off a dot on an alias.
//
// The alias comes along because it is part of the name. A shape of another module is never
// written without one, and two files reached through two aliases may both call a shape
// Square.
func nameAt(tokens []token.Token, j int) string {
	name := string(tokens[j].GetMatch())
	if j >= 2 && tokens[j-1].GetTag().Id == token.DOT && tokens[j-2].GetTag().Id == token.ID {
		return string(tokens[j-2].GetMatch()) + "." + name
	}
	return name
}

// claimedAt reads the shape name written after `as` or `returns`, and nothing when what
// follows is not one yet.
func claimedAt(tokens []token.Token, j int) string {
	if j >= len(tokens) || tokens[j].GetTag().Id != token.ID {
		return ""
	}
	if j+2 < len(tokens) && tokens[j+1].GetTag().Id == token.DOT && tokens[j+2].GetTag().Id == token.ID {
		return nameAt(tokens, j+2)
	}
	return string(tokens[j].GetMatch())
}

// fieldsBefore answers the fields offered at a position, when what sits in front of the
// cursor is a dot on a value whose shape is known. Anything else gives nothing, and the
// caller falls back to the ordinary list.
func (s shapeTable) fieldsBefore(tokens []token.Token, offset int) []string {
	owner := ownerBefore(tokens, offset)
	if owner == nil {
		return nil
	}
	return s.fields[s.reads[string(owner.GetMatch())]]
}

// ownerBefore answers the name a dot at this offset hangs off, and nil when there is no dot
// there. Both readers of a dot ask it — a shape to offer its fields, a module to say that
// what follows is not a field at all.
func ownerBefore(tokens []token.Token, offset int) token.Token {
	// The token being typed may already have started, so walk back over it first.
	i := len(tokens) - 1
	for ; i >= 0; i-- {
		if tokens[i].GetCursor() < offset {
			break
		}
	}
	if i < 0 {
		return nil
	}
	// `p.|` sits on the dot; `p.x|` sits on a name that follows one.
	if tokens[i].GetTag().Id == token.ID && i > 0 {
		i--
	}
	if i < 1 || tokens[i].GetTag().Id != token.DOT {
		return nil
	}

	owner := tokens[i-1]
	if owner.GetTag().Id != token.ID {
		return nil
	}
	return owner
}

// shapeAt describes the shape a token belongs to, for hover: the declaration itself, or a
// field being read out of a value.
func (s shapeTable) shapeAt(tokens []token.Token, subject token.Token) (string, []string, int) {
	name := string(subject.GetMatch())

	if fields, declared := s.fields[name]; declared {
		return name, fields, -1
	}

	// Tokens are compared by where they start: the concrete type holds slices and cannot be
	// compared with ==.
	for i, t := range tokens {
		if t.GetCursor() != subject.GetCursor() || i < 2 || tokens[i-1].GetTag().Id != token.DOT {
			continue
		}
		owner := tokens[i-2]
		if owner.GetTag().Id != token.ID {
			continue
		}
		shape := s.reads[string(owner.GetMatch())]
		fields := s.fields[shape]
		if index := slices.Index(fields, name); index >= 0 {
			return shape, fields, index
		}
	}

	return "", nil, -1
}

// fieldCompletions turns field names into items, with the index each one reads.
func fieldCompletions(fields []string) []CompletionItem {
	items := make([]CompletionItem, 0, len(fields))
	for index, field := range fields {
		items = append(items, CompletionItem{
			Label:  field,
			Detail: "field " + strconv.Itoa(index) + ", one tape wide",
			Kind:   Field,
		})
	}
	return items
}
