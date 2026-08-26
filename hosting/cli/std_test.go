package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// What comes with the language is read from where the toolchain installed it, and not from
// under the program.
//
// `std/` is the whole of what says so — the same way Go tells its own library from everything
// else by what the import path looks like. The second segment names the target: std/evm is
// written over the EVM's own instructions and means nothing anywhere else, so a module that
// cannot cross is marked as not crossing rather than found out at the far end.
func TestAModuleOfTheLanguageIsReadFromWhereItWasInstalled(t *testing.T) {
	root := stdRootOf(t)

	for _, tc := range []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "kept and read back",
			source: "use std/evm/storage as s;\ns.set(1, 42);\nprintd s.get(1);",
			want:   "42",
		},
		{
			// A set returns what it kept, so it stands where any other value does.
			name:   "a set is worth what it kept",
			source: "use std/evm/storage as s;\nprintd s.set(1, 41) + 1;",
			want:   "42",
		},
		{
			name:   "a key nothing was kept under",
			source: "use std/evm/storage as s;\ns.set(1, 42);\nprintd s.get(2);",
			want:   "0",
		},
		{
			// Read, add, keep — and inside the wrapper the reading needs no parentheses,
			// which is one of the things the wrapper is for.
			name: "read, added to, and kept again",
			source: "use std/evm/storage as s;\n" +
				"ident bump = defer { s.set(1, s.get(1) + feed(0)); };\n" +
				"s.set(1, 40);\nbump(2);\nprintd s.get(1);",
			want: "42",
		},
		{
			// The alias belongs to the file, the way it does for every other import.
			name:   "under an alias of the file's own",
			source: "use std/evm/storage as chain;\nchain.set(1, 42);\nprintd chain.get(1);",
			want:   "42",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projectOf(t, map[string]string{"src/main.ar": tc.source})
			printed, err := runWithStd(t, "src/main.ar", root)
			if err != nil {
				t.Fatalf("running: %v", err)
			}
			if got := strings.Fields(printed); len(got) == 0 || got[len(got)-1] != tc.want {
				t.Errorf("printed %q, want it to end with %s", printed, tc.want)
			}
		})
	}
}

// A module of the language that is not installed is refused where it was imported, naming the
// file it looked for — which is what tells somebody the toolchain is half installed rather
// than that they typed the name wrong.
func TestAModuleOfTheLanguageThatIsNotInstalled(t *testing.T) {
	projectOf(t, map[string]string{"src/main.ar": "use std/evm/nothing as n;\nprintd n.get(1);"})

	_, err := runWithStd(t, "src/main.ar", stdRootOf(t))
	if err == nil {
		t.Fatal("a module nobody installed was imported")
	}
	if !strings.Contains(err.Error(), "std/evm/nothing") {
		t.Errorf("error = %q, want it to name the module", err)
	}
}

// runWithStd runs a program that can reach what comes with the language.
func runWithStd(t *testing.T, entry, stdRoot string) (string, error) {
	t.Helper()

	var stdout bytes.Buffer
	err := newSession(t, sessionOpts{stdout: &stdout, stdRoot: stdRoot}).Run(t.Context(), filepath.FromSlash(entry))
	return stdout.String(), err
}

// stdRootOf answers the repository's own lib, so the test reads the same files `make
// install-std` copies. Reading them where they are written is what keeps this from passing
// against a stale copy somebody installed months ago.
func stdRootOf(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "lib"))
	if err != nil {
		t.Fatalf("finding the standard library: %v", err)
	}
	return root
}
