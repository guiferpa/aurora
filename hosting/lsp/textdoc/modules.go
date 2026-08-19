package textdoc

import (
	"slices"
	"strings"

	"github.com/guiferpa/aurora/wire/token"
)

// What the language server knows about the modules a document brings in, read straight from
// the tokens for the same reason the structs are: a document being edited is broken most of
// the time, and `x.` — the moment somebody wants to be told what is inside a module — never
// parses.
//
// It is what one file says and nothing more. Which names a module actually has is a question
// for whoever holds the other files, and this answers what it can without them: that a name
// is a module rather than a value, and which module it is.

// moduleAliases is what the use lines declared: the alias, and the module it names.
type moduleAliases map[string]string

// scanUses reads `use a/b/c as x;` out of a token chain.
func scanUses(tokens []token.Token) moduleAliases {
	found := make(moduleAliases)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].GetTag().Id != token.USE {
			continue
		}
		if alias, specifier, next := readUse(tokens, i); alias != "" {
			found[alias] = specifier
			i = next
		}
	}
	return found
}

// readUse reads one declaration starting at the use token: the path it names, and the alias
// it is reached by. A half-written line answers with nothing, which is the common case while
// somebody is typing one.
func readUse(tokens []token.Token, i int) (string, string, int) {
	specifier := ""
	for j := i + 1; j < len(tokens); j++ {
		switch tokens[j].GetTag().Id {
		case token.ID:
			specifier += string(tokens[j].GetMatch())
		case token.DIV:
			specifier += "/"
		case token.AS:
			if specifier == "" || j+1 >= len(tokens) || tokens[j+1].GetTag().Id != token.ID {
				return "", "", j
			}
			return string(tokens[j+1].GetMatch()), specifier, j + 1
		default:
			return "", "", j
		}
	}
	return "", "", len(tokens) - 1
}

// moduleBefore answers the module a dot at this offset reaches into, when what sits in front
// of it is an alias.
func (m moduleAliases) moduleBefore(tokens []token.Token, offset int) (string, bool) {
	owner := ownerBefore(tokens, offset)
	if owner == nil {
		return "", false
	}
	specifier, ok := m[string(owner.GetMatch())]
	return specifier, ok
}

// describe answers what hover has to say about a token, when what it has to say is about a
// module: the alias itself, or a name being reached through one.
func (m moduleAliases) describe(tokens []token.Token, subject token.Token) string {
	name := string(subject.GetMatch())
	if specifier, ok := m[name]; ok {
		return "module " + specifier + "\nreach what is inside it with " + name + ".name"
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
		if specifier, ok := m[string(owner.GetMatch())]; ok {
			return name + "\nof module " + specifier
		}
	}
	return ""
}

// moduleCompletions offers the aliases a document declared, said to be modules rather than
// values: what follows one is a dot, and never anything a value would take.
func moduleCompletions(aliases moduleAliases) []CompletionItem {
	items := make([]CompletionItem, 0, len(aliases))
	for alias, specifier := range aliases {
		items = append(items, CompletionItem{
			Label:  alias,
			Detail: "module " + specifier,
			Kind:   Module,
		})
	}
	slices.SortFunc(items, func(a, b CompletionItem) int {
		return strings.Compare(a.Label, b.Label)
	})
	return items
}
