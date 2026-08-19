package textdoc

import (
	"slices"
	"strings"

	"github.com/guiferpa/aurora/loader"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/module"
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

// scanUses reads every `use a/b/c as x;` out of a token chain, with the token of each so
// whatever goes wrong with one can be underlined where it was written.
func scanUses(tokens []token.Token) []ast.UseDeclaration {
	found := make([]ast.UseDeclaration, 0)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].GetTag().Id != token.USE {
			continue
		}
		if alias, specifier, next := readUse(tokens, i); alias != "" {
			found = append(found, ast.UseDeclaration{Specifier: specifier, Alias: alias, Token: tokens[i]})
			i = next
		}
	}
	return found
}

// aliasesOf is what those declarations say about names: the alias, and what it means.
func aliasesOf(uses []ast.UseDeclaration) moduleAliases {
	aliases := make(moduleAliases, len(uses))
	for _, declaration := range uses {
		aliases[declaration.Alias] = declaration.Specifier
	}
	return aliases
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

// exportCompletions offers what a module declared, which is read from the module's own tree.
//
// A module that could not be found offers nothing rather than everything: the diagnostic
// already says it is missing, and a list of keywords would be the one answer that is
// certainly wrong.
func exportCompletions(analysis *Analysis, specifier string) []CompletionItem {
	found, ok := analysis.module(specifier)
	if !ok {
		return []CompletionItem{}
	}

	names := loader.Exports(found)
	items := make([]CompletionItem, 0, len(names))
	for _, name := range names {
		items = append(items, CompletionItem{
			Label:  name,
			Detail: describeNode(exportedValue(found, name)) + " of module " + specifier,
			Kind:   Variable,
		})
	}
	return items
}

// describeExport says what a name reached through a module is, when the module is there to
// say — the same description an ordinary identifier gets, read from the other file.
func describeExport(analysis *Analysis, subject token.Token) string {
	specifier, ok := moduleOfSubject(analysis, subject)
	if !ok {
		return ""
	}
	found, loaded := analysis.module(specifier)
	if !loaded {
		return ""
	}
	value := exportedValue(found, string(subject.GetMatch()))
	if value == nil {
		return ""
	}
	return "\nvalue: " + describeNode(value)
}

// moduleOfSubject answers the module a token is reached through, when it follows a dot on an
// alias.
func moduleOfSubject(analysis *Analysis, subject token.Token) (string, bool) {
	aliases := aliasesOf(scanUses(analysis.Tokens))
	for i, t := range analysis.Tokens {
		if t.GetCursor() != subject.GetCursor() || i < 2 || analysis.Tokens[i-1].GetTag().Id != token.DOT {
			continue
		}
		specifier, ok := aliases[string(analysis.Tokens[i-2].GetMatch())]
		return specifier, ok
	}
	return "", false
}

// exportedValue is what a module bound a name to, or nil when it bound no such name.
func exportedValue(found module.Module, name string) ast.Node {
	for _, node := range found.Tree.Nodes {
		binding, ok := node.(ast.IdentLiteral)
		if ok && found.Symbol(binding.Id) == name {
			return binding.Value
		}
	}
	return nil
}
