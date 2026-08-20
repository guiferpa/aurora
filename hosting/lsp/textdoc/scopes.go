package textdoc

import "github.com/guiferpa/aurora/wire/token"

// Which declaration each name written in a file means.
//
// Go to definition asks this of one name and takes the nearest answer. Renaming asks it of
// every name at once and cannot approximate: an occurrence left behind is a program that
// stops compiling, and an occurrence taken by mistake is one that compiles and does the wrong
// thing. So the scopes are read as the language reads them.

// A scopeTable is every name a file writes, resolved: the declaration each occurrence means,
// and which declarations sit at the top of the file, where another file could reach them.
type scopeTable struct {
	// means is the cursor of an occurrence -> the token that declares it. A name nothing
	// declares is absent, which is the same answer a name of another file gets.
	means map[int]token.Token
	// tops is the cursor of a declaration -> whether another file could be writing it too.
	tops map[int]bool
}

// A binding is one name declared, and the stretch of the file it is a name in: the block it
// was written in, from its opening brace to its closing one, or the whole file.
type binding struct {
	name        string
	declaration token.Token
	at          int // where the declaration is written, as a token index
	from, to    int // the scope it belongs to, as token indexes
}

// scopesOf answers what every name written in a file means.
func scopesOf(tokens []token.Token) scopeTable {
	bindings := bindingsOf(tokens)
	table := scopeTable{means: make(map[int]token.Token), tops: make(map[int]bool)}

	for _, b := range bindings {
		// An alias is written at the top of a file and reaches no further than it: it names
		// a module for this file alone, and no other file can write it.
		_, alias := aliasOfUse(tokens, b.declaration)
		table.tops[b.declaration.GetCursor()] = b.from == 0 && b.to == len(tokens) && !alias
	}

	for i, tk := range tokens {
		if tk.GetTag().Id != token.ID || !occurrence(tokens, i) {
			continue
		}
		if found := meaning(bindings, string(tk.GetMatch()), i); found != nil {
			table.means[tk.GetCursor()] = found
		}
	}
	return table
}

// bindingsOf reads every declaration a file makes, with the scope each one is a name in.
//
// The scope is the block it was written in rather than everything after it, because that is
// what the language does: a name declared inside a block is gone at its closing brace. A
// block left open by somebody still typing runs to the end of the file, which is what it will
// mean once they close it.
func bindingsOf(tokens []token.Token) []binding {
	found := make([]binding, 0)
	blocks := make([]int, 0) // the braces open right now, as token indexes
	closes := make(map[int]int)
	inside := make([]int, 0) // for each binding, the brace it was written inside, or -1

	for i, tk := range tokens {
		switch tk.GetTag().Id {
		case token.O_CUR_BRK:
			blocks = append(blocks, i)
		case token.C_CUR_BRK:
			if len(blocks) > 0 {
				closes[blocks[len(blocks)-1]] = i
				blocks = blocks[:len(blocks)-1]
			}
		case token.ID:
			if !declaredHere(tokens, i) {
				continue
			}
			block := -1
			if len(blocks) > 0 {
				block = blocks[len(blocks)-1]
			}
			found = append(found, binding{name: string(tk.GetMatch()), declaration: tk, at: i})
			inside = append(inside, block)
		}
	}

	for k, block := range inside {
		if block < 0 {
			found[k].from, found[k].to = 0, len(tokens)
			continue
		}
		found[k].from = block
		if closing, closed := closes[block]; closed {
			found[k].to = closing
		} else {
			found[k].to = len(tokens)
		}
	}
	return found
}

// meaning answers which declaration a name written at i means: the innermost scope that holds
// both, and inside one scope the nearest declaration above it.
//
// Above it, because a name bound twice in the same scope is two names and the second shadows
// the first from where it is written. When there is none above, the one below answers: a
// deferred scope runs when it is called, so its body may name something written under it.
func meaning(bindings []binding, name string, at int) token.Token {
	var chosen *binding
	for i := range bindings {
		b := &bindings[i]
		if b.name != name || at < b.from || at > b.to {
			continue
		}
		if chosen == nil || closer(b, chosen, at) {
			chosen = b
		}
	}
	if chosen == nil {
		return nil
	}
	return chosen.declaration
}

// closer answers whether one binding is the one a name at this point means, against another
// that also holds it: the deeper scope first, then the declaration above rather than below,
// then the nearest of the two.
func closer(candidate, chosen *binding, at int) bool {
	if candidate.from != chosen.from {
		return candidate.from > chosen.from
	}
	if (candidate.at <= at) != (chosen.at <= at) {
		return candidate.at <= at
	}
	if candidate.at <= at {
		return candidate.at > chosen.at
	}
	return candidate.at < chosen.at
}

// declaredHere answers whether the name at i is being declared rather than used: `ident x`,
// `shape Point`, and the alias of a use line.
//
// `as` is the alias only inside a use line — everywhere else it claims a shape, and the name
// after it is one being used.
func declaredHere(tokens []token.Token, i int) bool {
	if i == 0 {
		return false
	}
	switch tokens[i-1].GetTag().Id {
	case token.IDENT, token.SHAPE:
		return true
	case token.AS:
		_, inUse := aliasOfUse(tokens, tokens[i])
		return inUse
	}
	return false
}

// occurrence answers whether the name at i is one this file declares somewhere.
//
// Three names are written that are not: a field, which belongs to a shape rather than to a
// scope; a name reached through an alias, which belongs to another file; and the path of a
// use line, which names a module and not a value.
func occurrence(tokens []token.Token, i int) bool {
	if i > 0 && tokens[i-1].GetTag().Id == token.DOT {
		return false
	}
	if insideShapeDeclaration(tokens, i) {
		return false
	}
	_, inUse := aliasOfUse(tokens, tokens[i])
	return !inUse
}
