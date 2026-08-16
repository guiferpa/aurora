package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/guiferpa/aurora/manifest"
)

// InitManifestTemplate is the template for aurora.toml. Use fmt.Sprintf with projectName for the name field.
const InitManifestTemplate = `# Aurora project manifest.
# See https://github.com/guiferpa/aurora for more information.

[project]
# Project identifier (inherited from the root folder name where 'aurora init' was run).
name = %q
# Project version (semantic version recommended).
version = "0.1.0"

[profiles.main]
# Default profile. Commands like 'aurora build' or 'aurora run' use these paths when no file is given.
# Path to the main source file. Used by build, run, and deploy when no file argument is passed.
source = "src/main.ar"
# Path where the compiled binary is written. Name matches the source filename (without extension). Used by 'aurora build' when no -o output is passed.
binary = "bin/main"
# Bytes per value. Every value is a tape this wide, text included — and the greeting in
# src/main.ar is eleven bytes, which does not fit the default eight.
tape_size = 16
`

// InitSourceTemplate is the program a new project starts from.
//
// "Abidu abide" is what Aurora — the language is named after the author's daughter — was
// saying at one year old. It felt like the right first thing for the language to say.
const InitSourceTemplate = `#- Welcome to Aurora.
#-
#- Run this with:
#-
#-   aurora run     the main profile, which points here
#-   aurora test    the assertions in main.test.ar
#-
#- Every value is a tape: a fixed run of bytes, 8 by default. There are three
#- ways to read one — printb for its bytes, printd for its number, printc for
#- the characters it names.

ident greet = defer { "Abidu abide"; };

printc greet();
`

// InitTestTemplate is the test that comes with a new project.
const InitTestTemplate = `#- Tests for main.ar.
#-
#- A test belongs to the source file of the same name: this file tests main.ar,
#- which runs first, so greet is in scope here without being imported.
#-
#- Text is a tape holding its bytes, so two texts are equal when their bytes are
#- — no special rule for comparing them.
#-
#- Run them with:
#-
#-   aurora test

assert(greet() equals "Abidu abide", "greet says its piece");
`

// InitInput is the input for the Init handler.
type InitInput struct {
	Dir         string // directory to write the project into (usually cwd)
	ProjectName string // default: filepath.Base(Dir)
	Stdout      io.Writer
}

// Init starts a project: the manifest, a program to run, and a test for it.
//
// A manifest on its own leaves someone with nothing to run and no clue where to put it, so
// the layout it describes comes with it — src/main.ar and src/main.test.ar, which are what
// the main profile points at.
func Init(in InitInput) error {
	if in.Dir == "" {
		return fmt.Errorf("InitInput.Dir is required")
	}

	manifestPath := filepath.Join(in.Dir, manifest.Filename)
	if _, err := os.Stat(manifestPath); err == nil {
		return fmt.Errorf("%s already exists", manifest.Filename)
	}

	name := in.ProjectName
	if name == "" {
		name = filepath.Base(in.Dir)
	}

	sourceDir := filepath.Join(in.Dir, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return err
	}

	files := []struct {
		path    string
		content string
	}{
		{path: manifestPath, content: fmt.Sprintf(InitManifestTemplate, name)},
		{path: filepath.Join(sourceDir, "main.ar"), content: InitSourceTemplate},
		{path: filepath.Join(sourceDir, "main.test.ar"), content: InitTestTemplate},
	}

	created := make([]string, 0, len(files))
	for _, file := range files {
		// A source file already sitting there belongs to whoever put it there.
		if _, err := os.Stat(file.path); err == nil {
			continue
		}
		if err := os.WriteFile(file.path, []byte(file.content), 0o644); err != nil {
			return err
		}
		created = append(created, file.path)
	}

	report(in.Stdout, in.Dir, created)
	return nil
}

func report(w io.Writer, dir string, created []string) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintln(w, "✨ dawn has broken on your project.")
	for _, path := range created {
		if relative, err := filepath.Rel(dir, path); err == nil {
			path = relative
		}
		_, _ = fmt.Fprintf(w, "   %s\n", path)
	}
	_, _ = fmt.Fprintln(w, "\nRun it with 'aurora run', test it with 'aurora test'.")
}
