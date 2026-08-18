package textdoc

import (
	"slices"
	"strconv"

	"github.com/guiferpa/aurora/wire/token"
)

// What the language server knows about the struct declarations, read straight from the tokens
// rather than from the tree.
//
// It has to be the tokens. A document being edited is broken most of the time — the moment
// someone types `p.` there is no field name yet and nothing parses — and completion is
// exactly what is wanted at that moment. The parser drops its own tables when the parse
// ends, and would have nothing to offer here anyway.
type structShapes struct {
	fields map[string][]string // struct name -> its fields, in order
	shapes map[string]string   // identifier name -> the struct it is read as
}

// scanStructs reads the declarations out of a token stream: `struct Point { x, y }` for the
// fields, and `ident p = Point{...}` or `... as Point` for what a name is read as.
func scanStructs(tokens []token.Token) structShapes {
	found := structShapes{
		fields: make(map[string][]string),
		shapes: make(map[string]string),
	}

	for i := 0; i < len(tokens); i++ {
		switch tokens[i].GetTag().Id {
		case token.STRUCT:
			if name, fields, next := readDeclaration(tokens, i); name != "" {
				found.fields[name] = fields
				i = next
			}
		case token.IDENT:
			// `ident p = Point{` and `ident p = <anything> as Point` both say what p is.
			if name, shape := readBinding(tokens, i); shape != "" {
				found.shapes[name] = shape
			}
		}
	}

	return found
}

// readDeclaration reads `struct Name { a, b }` starting at the struct token.
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

// readBinding reads what `ident name = ...` binds, up to the end of the statement.
func readBinding(tokens []token.Token, i int) (string, string) {
	if i+2 >= len(tokens) || tokens[i+1].GetTag().Id != token.ID || tokens[i+2].GetTag().Id != token.ASSIGN {
		return "", ""
	}
	name := string(tokens[i+1].GetMatch())

	for j := i + 3; j < len(tokens); j++ {
		switch tokens[j].GetTag().Id {
		case token.SEMICOLON:
			return name, ""
		case token.AS:
			// The nearest `as` wins, which is the one the value ends up with.
			if j+1 < len(tokens) && tokens[j+1].GetTag().Id == token.ID {
				return name, string(tokens[j+1].GetMatch())
			}
		case token.ID:
			// A construction: the name of a struct in front of a brace.
			if j+1 < len(tokens) && tokens[j+1].GetTag().Id == token.O_CUR_BRK {
				return name, string(tokens[j].GetMatch())
			}
		}
	}
	return name, ""
}

// fieldsBefore answers the fields offered at a position, when what sits in front of the
// cursor is a dot on a value whose shape is known. Anything else gives nothing, and the
// caller falls back to the ordinary list.
func (s structShapes) fieldsBefore(tokens []token.Token, offset int) []string {
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
	return s.fields[s.shapes[string(owner.GetMatch())]]
}

// structAt describes the struct a token belongs to, for hover: the declaration itself, or a
// field being read out of a value.
func (s structShapes) structAt(tokens []token.Token, subject token.Token) (string, []string, int) {
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
		shape := s.shapes[string(owner.GetMatch())]
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
