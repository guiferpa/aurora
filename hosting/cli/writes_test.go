package cli

import (
	"context"
	"strings"
	"testing"
)

// Whether reaching a scope changes anything the chain keeps, which is what decides between a
// question and a change.
//
// It has to follow calls. Once there is a standard library, the scope somebody writes holds no
// sstore at all — `s.set(...)` is a call, and the write is one file away.
func TestWhetherReachingAScopeChangesAnything(t *testing.T) {
	const source = "use std/evm/storage as s;\n" +
		"ident read = defer { s.get(1); };\n" +
		"ident keep = defer { s.set(1, feed(0)); };\n" +
		"ident through = defer { keep(feed(0)); };\n" +
		"ident deeper = defer { through(feed(0)); };\n" +
		"ident sums = defer { feed(0) + feed(1); };\n" +
		"ident raw = defer { sstore 1 feed(0); };\n"

	projectOf(t, map[string]string{"src/main.ar": source})
	blocks, err := newSession(t, sessionOpts{}).Blocks("src/main.ar")
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}

	for _, tc := range []struct {
		name   string
		writes bool
	}{
		{name: "read", writes: false},
		{name: "sums", writes: false},
		{name: "raw", writes: true},
		{name: "keep", writes: true},
		// The two that matter: the write is not in this scope at all.
		{name: "through", writes: true},
		{name: "deeper", writes: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes, found := ScopeWrites(WritesInput{Blocks: blocks, Function: tc.name})
			if !found {
				t.Fatalf("%s is not a scope of the program", tc.name)
			}
			if writes != tc.writes {
				t.Errorf("%s writes = %v, want %v", tc.name, writes, tc.writes)
			}
		})
	}

	t.Run("a name the program does not bind", func(t *testing.T) {
		if _, found := ScopeWrites(WritesInput{Blocks: blocks, Function: "nothing"}); found {
			t.Error("it found a scope nobody bound")
		}
	})
}

// A call over a scope that keeps something is refused before a chain is spoken to.
//
// This is the failure it exists for, and it happened on a real one: a scope that read a
// counter, added one and kept it answered 1 every time, because every call started from the
// state the last one did not change. Nothing was wrong with the bytecode. The question was the
// wrong question.
func TestACallOverAScopeThatKeepsSomethingIsRefused(t *testing.T) {
	const source = "use std/evm/storage as s;\n" +
		"ident inc = defer { s.set(1, s.get(1) + 1); };\n" +
		"ident read = defer { s.get(1); };\n"

	projectOf(t, map[string]string{"src/main.ar": source})
	blocks, err := newSession(t, sessionOpts{}).Blocks("src/main.ar")
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}

	// No RPC is dialled, because it never gets that far.
	err = Call(context.Background(), CallInput{Function: "inc", Blocks: blocks, RPC: "http://127.0.0.1:1"})
	if err == nil {
		t.Fatal("it asked a question of something that changes the answer")
	}
	if !strings.Contains(err.Error(), "aurora tx inc") {
		t.Errorf("it says %q, and never says what to type instead", err)
	}
	if !strings.Contains(err.Error(), "changes what the chain keeps") {
		t.Errorf("it says %q, and never says why", err)
	}
}

// And a call over one that keeps nothing is not refused: it gets as far as the network, which
// is where this test stops caring.
func TestACallOverAScopeThatKeepsNothingIsNotRefused(t *testing.T) {
	const source = "use std/evm/storage as s;\nident read = defer { s.get(1); };\n"

	projectOf(t, map[string]string{"src/main.ar": source})
	blocks, err := newSession(t, sessionOpts{}).Blocks("src/main.ar")
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}

	err = Call(context.Background(), CallInput{Function: "read", Blocks: blocks, RPC: "http://127.0.0.1:1"})
	if err == nil {
		t.Fatal("it reached a network that is not there")
	}
	if strings.Contains(err.Error(), "aurora tx") {
		t.Errorf("it refused a question worth asking: %q", err)
	}
}

// With no program in hand, nothing is refused. A contract was deployed from a source that may
// since have changed, or moved, or belong to somebody else's checkout — not knowing is a
// reason to say less, never a reason to stop somebody reaching what is already out there.
func TestWithNoProgramNothingIsRefused(t *testing.T) {
	err := Call(context.Background(), CallInput{Function: "inc", RPC: "http://127.0.0.1:1"})
	if err != nil && strings.Contains(err.Error(), "aurora tx") {
		t.Errorf("it refused without knowing: %q", err)
	}
}
