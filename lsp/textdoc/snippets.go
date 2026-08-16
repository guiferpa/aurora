package textdoc

import (
	"strconv"
	"strings"

	"github.com/guiferpa/aurora/lexer"
)

// What a keyword expands to when the client expands snippets.
//
// Every one of these is a form someone would otherwise type out and get wrong the first few
// times: where the semicolon goes, that a branch ends in a fallback with no test, that the
// index of head and tail is a literal number. The placeholders name what goes there rather
// than saying "1", so the shape reads as a sentence before it is filled in.
//
// A keyword with no entry is offered as itself.
var keywordSnippets = map[string]string{
	lexer.IDENT:  "ident ${1:name} = ${0:value};",
	lexer.DEFER:  "defer {\n\t$0\n}",
	lexer.IF:     "if ${1:condition} {\n\t$0\n}",
	lexer.BRANCH: "branch {\n\t${1:test}: ${2:value},\n\t${0:fallback};\n}",
	lexer.STRUCT: "struct ${1:Name} { ${0:field} };",
	lexer.AS:     "as ${0:Struct}",
	lexer.ASSERT: "assert(${1:condition}, \"${0:message}\");",
	lexer.FEED:   "feed(${0:0})",
	lexer.PRINTB: "printb ${0:value};",
	lexer.PRINTC: "printc ${0:value};",
	lexer.PRINTD: "printd ${0:value};",
	// The tape operations take a target and then a value; for head and tail that value is
	// an index, and it has to be written as a number.
	lexer.PULL: "pull ${1:tape} ${0:value}",
	lexer.PUSH: "push ${1:tape} ${0:value}",
	lexer.HEAD: "head ${1:tape} ${0:1}",
	lexer.TAIL: "tail ${1:tape} ${0:1}",
}

// keywordCompletion offers a keyword, as a snippet when there is one and the client expands
// them.
func keywordCompletion(tag lexer.Tag, snippets bool) CompletionItem {
	item := CompletionItem{
		Label:  tag.Keyword,
		Detail: tag.Description,
		Kind:   Keyword,
	}

	if body, ok := keywordSnippets[tag.Id]; ok && snippets {
		item.InsertText = body
		item.InsertTextFormat = SnippetFormat
	}
	return item
}

// structCompletions offers each declared struct as a way of building one, with the field
// names as the places to fill in — the directive already said what they are.
//
//	struct Point { x, y };   ->   Point{${1:x}, ${2:y}}
func structCompletions(shapes structShapes, snippets bool) []CompletionItem {
	items := make([]CompletionItem, 0, len(shapes.fields))
	for name, fields := range shapes.fields {
		item := CompletionItem{
			Label:  name,
			Detail: "struct: " + strings.Join(fields, ", "),
			Kind:   Struct,
		}
		if snippets {
			item.InsertText = name + "{" + placeholders(fields) + "}"
			item.InsertTextFormat = SnippetFormat
		}
		items = append(items, item)
	}
	return items
}

// placeholders turns field names into the numbered stops a snippet is filled in through.
func placeholders(fields []string) string {
	out := make([]string, 0, len(fields))
	for i, field := range fields {
		out = append(out, "${"+strconv.Itoa(i+1)+":"+field+"}")
	}
	return strings.Join(out, ", ")
}
