package cli

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The documentation is full of Aurora, and prose rots quietly: a snippet keeps saying what
// the language used to do long after it stopped. Reading for that does not work — three
// snippets were still describing reels the day text became a tape, and they were only found
// by running them.
//
// So they are run. Every fenced ```aurora block in the language documentation is executed,
// and a block says which of three things it is, in its own text:
//
//	#- greeting.ar          a named file of a multi-file example; the files run together
//	#- fails: <message>     a block that documents an error; it must fail, and say that
//	(neither)               a whole program; it must run
//
// A block written to be read rather than run — a grammar rule, a form with a placeholder —
// is left alone.

var (
	// auroraBlock matches a fenced block written in Aurora.
	auroraBlock = regexp.MustCompile("(?s)```aurora\n(.*?)```")
	// namedFile matches the comment a multi-file example opens each block with.
	namedFile = regexp.MustCompile(`^#-\s+(\S+\.ar)\s*$`)
	// expectedFailure matches the marker a block documenting an error carries.
	expectedFailure = regexp.MustCompile(`#-\s*fails:\s*([^\n—]+)`)
)

// snippet is one Aurora block of a document.
type snippet struct {
	code string
	line int
	// file is the name the block gave itself, when it belongs to a multi-file example.
	file string
	// wantErr is the message the block says it fails with, when it documents an error.
	wantErr string
}

// snippetsOf reads the Aurora blocks of a document, skipping the ones written to be read
// rather than run: a grammar rule (`_expr -> …`) or a form with a placeholder
// (`assert(<condition>, …)`).
func snippetsOf(source string) []snippet {
	snippets := make([]snippet, 0)

	for _, match := range auroraBlock.FindAllStringSubmatchIndex(source, -1) {
		code := source[match[2]:match[3]]
		if strings.Contains(code, "->") || strings.Contains(code, "<") {
			continue
		}

		s := snippet{code: code, line: strings.Count(source[:match[0]], "\n") + 1}
		if named := namedFile.FindStringSubmatch(strings.SplitN(code, "\n", 2)[0]); named != nil {
			s.file = named[1]
		}
		if fails := expectedFailure.FindStringSubmatch(code); fails != nil {
			s.wantErr = strings.TrimSpace(fails[1])
		}
		snippets = append(snippets, s)
	}
	return snippets
}

// documentationFiles lists the markdown expected to hold runnable Aurora.
//
// A document that opens by declaring itself a proposal is skipped whole: it describes a
// language that does not exist yet, and the compiler is meant to reject its examples.
func documentationFiles(t *testing.T) []string {
	t.Helper()

	var docs []string
	for _, root := range []string{filepath.Join("..", ".."), filepath.Join("..", "..", "docs")} {
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatalf("reading %s: %v", root, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			path := filepath.Join(root, entry.Name())
			if isProposal(t, path) {
				continue
			}
			docs = append(docs, path)
		}
	}
	return docs
}

// isProposal answers whether a document declares itself one, which it does at the top.
func isProposal(t *testing.T, path string) bool {
	t.Helper()

	bs, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	head := string(bs)
	if len(head) > 800 {
		head = head[:800]
	}
	return strings.Contains(head, "Status: proposal")
}

// runSnippet runs one standalone block and checks it did what it said it would.
func runSnippet(t *testing.T, doc string, s snippet) {
	t.Helper()

	entry := filepath.Join(t.TempDir(), "main.ar")
	if err := os.WriteFile(entry, []byte(s.code), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Run(t.Context(), RunInput{Source: entry, Stdout: io.Discard})

	if s.wantErr == "" {
		if err != nil {
			t.Errorf("%s:%d does not run: %v\n%s", doc, s.line, err, s.code)
		}
		return
	}
	// The block says it fails, so passing is the failure — and the message it promises has
	// to be the message it gets.
	if err == nil {
		t.Errorf("%s:%d says it fails with %q, and it ran\n%s", doc, s.line, s.wantErr, s.code)
		return
	}
	if !strings.Contains(err.Error(), s.wantErr) {
		t.Errorf("%s:%d says it fails with %q, and it failed with %q", doc, s.line, s.wantErr, err)
	}
}

// runNamedExample writes the named blocks of a document side by side and runs the test files
// among them, which is what the pairing rule of `aurora test` does.
func runNamedExample(t *testing.T, doc string, named []snippet) {
	t.Helper()

	dir := t.TempDir()
	var tests []string
	for _, s := range named {
		path := filepath.Join(dir, s.file)
		if err := os.WriteFile(path, []byte(s.code), 0o644); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(s.file, TestExtension) {
			tests = append(tests, path)
		}
	}

	for _, path := range tests {
		report, err := Test(t.Context(), TestInput{Target: path, Stdout: io.Discard})
		if err != nil {
			t.Errorf("%s: the example named %s does not run: %v", doc, filepath.Base(path), err)
			continue
		}
		if !report.OK() {
			t.Errorf("%s: the example named %s does not pass: %d failed", doc, filepath.Base(path), report.Failed)
		}
		if report.Passed == 0 {
			t.Errorf("%s: the example named %s asserts nothing", doc, filepath.Base(path))
		}
	}
}

func TestDocumentationSnippetsRun(t *testing.T) {
	docs := documentationFiles(t)
	if len(docs) == 0 {
		t.Fatal("no documentation found, so nothing was checked")
	}

	checked := 0
	for _, doc := range docs {
		bs, err := os.ReadFile(doc)
		if err != nil {
			t.Fatalf("reading %s: %v", doc, err)
		}

		var named []snippet
		for _, s := range snippetsOf(string(bs)) {
			checked++
			if s.file != "" {
				named = append(named, s)
				continue
			}
			t.Run(filepath.Base(doc)+":"+strconv.Itoa(s.line), func(t *testing.T) {
				runSnippet(t, doc, s)
			})
		}

		if len(named) > 0 {
			t.Run(filepath.Base(doc)+" (named files)", func(t *testing.T) {
				runNamedExample(t, doc, named)
			})
		}
	}

	if checked == 0 {
		t.Fatal("no runnable snippet was found, so nothing was checked")
	}
}

// A document holding Aurora that nobody runs is the failure this file exists to prevent,
// and it arrives quietly: someone writes docs/contributing/foo.md with a snippet in it, and
// the walk above never looks there.
//
// So the walk is checked too.
func TestNoDocumentWithAuroraEscapesTheCheck(t *testing.T) {
	checked := make(map[string]bool)
	for _, doc := range documentationFiles(t) {
		absolute, err := filepath.Abs(doc)
		if err != nil {
			t.Fatal(err)
		}
		checked[absolute] = true
	}

	err := filepath.WalkDir(filepath.Join("..", ".."), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return err
		}
		if strings.Contains(path, "node_modules") || strings.Contains(path, ".git/") {
			return nil
		}

		bs, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !auroraBlock.Match(bs) || isProposal(t, path) {
			return nil
		}

		absolute, absErr := filepath.Abs(path)
		if absErr != nil {
			return absErr
		}
		if !checked[absolute] {
			t.Errorf("%s holds Aurora and nothing runs it: add it to documentationFiles, or declare it a proposal", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the repository: %v", err)
	}
}
