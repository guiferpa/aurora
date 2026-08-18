// Package trace shows what a compilation phase produced, for whoever asked to see it with
// -l/--loggers.
//
// It lives on the host side because a phase returns values and does not write: printing is
// a decision about someone's terminal, and a phase has no business making it. What each
// phase hands back — the tree, the instructions — is enough to show it afterwards, so
// nothing had to be threaded into the phases for this to work.
//
// What is lost by printing here rather than inside is that the output is no longer
// on-time: it appears once a phase has finished, not while it runs.
package trace

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"

	"github.com/fatih/color"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/wire/ast"
	"github.com/guiferpa/aurora/wire/ir"
	"github.com/guiferpa/aurora/wire/token"
)

// Tokens writes what the lexer read, one tk per line: the tag, then the bytes it
// matched.
func Tokens(w io.Writer, tokens []token.Token) error {
	for _, tk := range tokens {
		id := color.New(color.FgHiCyan).Sprint(tk.GetTag().Id)
		match := color.New(color.FgHiYellow).Sprint(byteutil.ToHexBloom(tk.GetMatch()))
		if _, err := fmt.Fprintf(w, "%s: %s\n", id, match); err != nil {
			return err
		}
	}
	return nil
}

// AST writes the tree the parser produced, as JSON.
func AST(w io.Writer, ast ast.AST) error {
	bs, err := json.MarshalIndent(wrapNode(ast), "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, colorizeJSON(string(bs)))
	return err
}

// Instructions writes the IR the emitter produced, one instruction per line.
func Instructions(w io.Writer, insts []ir.Instruction) error {
	_, err := fmt.Fprintln(w, ir.Format(insts))
	return err
}

// node is one node of the tree, as it is shown: the type name, then its fields.
//
// The name is what makes the JSON readable — a tree of anonymous objects says nothing about
// which node is which, and the type is the first thing anyone reading a tree looks for.
type node struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

var nodeType = reflect.TypeOf((*ast.Node)(nil)).Elem()

// wrapNode walks a node and labels every node inside it with its type.
//
// It is reflection because the alternative is a method on every node type in the parser,
// which would put the shape of this output inside the tree it describes.
func wrapNode(n any) any {
	if n == nil {
		return nil
	}

	v := reflect.ValueOf(n)
	t := reflect.TypeOf(n)

	// A slice of nodes: each one is wrapped.
	if t.Kind() == reflect.Slice && t.Elem().Implements(nodeType) {
		length := v.Len()
		wrapped := make([]any, 0, length)
		for i := 0; i < length; i++ {
			wrapped = append(wrapped, wrapNode(v.Index(i).Interface()))
		}
		return wrapped
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		v = v.Elem()
		if !v.IsValid() {
			return nil
		}
	}

	fields := make(map[string]any)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// An unexported field cannot be read through reflection at all — asking panics — and
		// it has nothing to show anyway: what a reader wants is what the node holds.
		if !field.IsExported() {
			continue
		}
		value := v.Field(i).Interface()

		switch value := value.(type) {
		case ast.Node:
			fields[field.Tag.Get("json")] = wrapNode(value)

		case []ast.Node:
			wrapped := make([]any, 0, len(value))
			for _, n := range value {
				wrapped = append(wrapped, wrapNode(n))
			}
			fields[field.Tag.Get("json")] = wrapped

		default:
			// A field tagged "-" is carried for the compiler and not for a reader: a tk
			// holds its whole source position, which drowns the tree it belongs to.
			if tag := field.Tag.Get("json"); tag != "-" && tag != "" {
				fields[tag] = value
			}
		}
	}

	return node{Type: t.Name(), Data: fields}
}

var (
	colorizeKey    = color.New(color.FgHiCyan).SprintFunc()
	colorizeString = color.New(color.FgHiYellow).SprintFunc()
	colorizeValue  = color.New(color.FgHiMagenta).SprintFunc()
)

// colorizeJSON paints keys, strings and scalars so a large tree can be skimmed.
func colorizeJSON(s string) string {
	keys := regexp.MustCompile(`"([^"]+)"\s*:`)
	s = keys.ReplaceAllString(s, colorizeKey(`"$1"`)+":")

	strings := regexp.MustCompile(`:\s*"([^"]*)"`)
	s = strings.ReplaceAllString(s, ": "+colorizeString(`"$1"`))

	scalars := regexp.MustCompile(`:\s*(\d+|true|false|null)`)
	s = scalars.ReplaceAllString(s, ": "+colorizeValue(`$1`))

	return s
}
