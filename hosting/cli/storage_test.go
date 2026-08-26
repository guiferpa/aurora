package cli

import (
	"strings"
	"testing"
)

// What the chain keeps between transactions, reached under an alias the file chose.
//
// It is written as an import and is not one: there is no src/storage.ar, nothing is resolved,
// and nothing crosses. What it is to somebody using it is exactly an import.
func TestStorageKeepsWhatWasWrittenUnderAKey(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "written and read back",
			source: "use storage as s;\ns.set(1, 42);\nprintd s.get(1);",
			want:   "42",
		},
		{
			// A write is an expression like everything else, so it is worth what it wrote.
			name:   "a write is worth what it wrote",
			source: "use storage as s;\nprintd s.set(1, 41) + 1;",
			want:   "42",
		},
		{
			name:   "a key nothing was written under reads as nothing",
			source: "use storage as s;\ns.set(1, 42);\nprintd s.get(2);",
			want:   "0",
		},
		{
			name:   "written over",
			source: "use storage as s;\ns.set(1, 42);\ns.set(1, 7);\nprintd s.get(1);",
			want:   "7",
		},
		{
			// The keys are values the program works out, which is what makes it a map rather
			// than a fixed set of places.
			name:   "a key a program worked out",
			source: "use storage as s;\nident k = 3 * 4;\ns.set(k, 42);\nprintd s.get(12);",
			want:   "42",
		},
		{
			// A scope reaches storage the way it reaches anything the language does, and this
			// is the shape a contract actually has: one scope writes, another reads.
			name: "a scope writes and another reads",
			source: "use storage as s;\n" +
				"ident put = defer { s.set(1, feed(0)); };\n" +
				"ident got = defer { s.get(1); };\n" +
				"put(9);\nprintd got();",
			want: "9",
		},
		{
			name:   "the alias is the file's own",
			source: "use storage as db;\ndb.set(1, 42);\nprintd db.get(1);",
			want:   "42",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projectOf(t, map[string]string{"src/main.ar": tc.source})
			printed, err := run(t, "src/main.ar")
			if err != nil {
				t.Fatalf("running: %v", err)
			}
			if got := strings.Fields(printed); len(got) == 0 || got[len(got)-1] != tc.want {
				t.Errorf("printed %q, want it to end with %s", printed, tc.want)
			}
		})
	}
}

// What is refused, and where it is said.
func TestWhatStorageRefuses(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{
			name:   "a name storage does not have",
			source: "use storage as s;\nprintd s.remove(1);",
			want:   "storage has no remove",
		},
		{
			name:   "a set with no value",
			source: "use storage as s;\ns.set(1);",
			want:   "s.set takes a key and a value",
		},
		{
			name:   "a get with too much",
			source: "use storage as s;\nprintd s.get(1, 2);",
			want:   "s.get takes a key",
		},
		{
			// The message says the word the file used, not the word the language uses.
			name:   "and it says what the file called it",
			source: "use storage as db;\ndb.set(1);",
			want:   "db.set takes a key and a value",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projectOf(t, map[string]string{"src/main.ar": tc.source})
			_, err := run(t, "src/main.ar")
			if err == nil {
				t.Fatal("expected a compile error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to say %q", err, tc.want)
			}
		})
	}
}
