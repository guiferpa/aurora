package cli

import (
	"strings"
	"testing"
)

// The two instructions that reach what a chain keeps between one transaction and the next.
//
// They are the machine's, not a library's — the EVM opcodes under the names they have there —
// and what makes them usable is a module written over them. That module is next; this is what
// it will be written over.
func TestWhatTheChainKeepsBetweenTransactions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "kept and read back",
			source: "sstore 1 42;\nprintd sload 1;",
			want:   "42",
		},
		{
			// A write is an expression like everything else, so it is worth what it kept.
			name:   "a write is worth what it kept",
			source: "printd sstore 1 41 + 1;",
			want:   "42",
		},
		{
			name:   "a key nothing was kept under reads as nothing",
			source: "sstore 1 42;\nprintd sload 2;",
			want:   "0",
		},
		{
			name:   "kept over",
			source: "sstore 1 42;\nsstore 1 7;\nprintd sload 1;",
			want:   "7",
		},
		{
			// The keys are values a program works out, which is what makes it a map rather
			// than a fixed set of places.
			name:   "a key a program worked out",
			source: "ident k = 3 * 4;\nsstore k 42;\nprintd sload 12;",
			want:   "42",
		},
		{
			// Read, add, keep: the shape every contract that counts something has.
			//
			// The parentheses are not decoration. Both of these take what follows them the way
			// printd does — as much of it as is an expression — so `sload 1 + 2` reads the key
			// three, and the sum has to be put outside the read.
			name:   "read, added to, and kept again",
			source: "sstore 1 40;\nsstore 1 ((sload 1) + 2);\nprintd sload 1;",
			want:   "42",
		},
		{
			// Said out loud, because somebody will write it: a read takes as much as it can,
			// which is the same rule every other instruction written this way follows.
			name:   "a read takes as much as follows it",
			source: "sstore 3 42;\nprintd sload 1 + 2;",
			want:   "42",
		},
		{
			// A scope reaches them like it reaches anything, which is what lets a module wrap
			// them: one scope writes, another reads, and the chain is what is between.
			name: "a scope keeps and another reads",
			source: "ident put = defer { sstore 1 feed(0); };\n" +
				"ident got = defer { sload 1; };\n" +
				"put(9);\nprintd got();",
			want: "9",
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
