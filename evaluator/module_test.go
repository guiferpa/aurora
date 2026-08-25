package evaluator

import (
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

// A program of several files, in memory.
//
// Each module is compiled on its own and appended to one stream, and each is run as a range
// of it. That is what the loader will do, written out here so the evaluator can be asked the
// questions only a program of several modules raises — and the stream is never sliced,
// because a call crossing a module lands on a block that has to be among the blocks the
// evaluator is holding.

// file is one module of a program: the name it is known by, and what is in it. The empty name
// is the file somebody asked to run.
type file struct {
	id     string
	source string
}

// decimals collects what a program printed, as numbers.
type decimals struct{ printed *[]uint64 }

func (d decimals) Print(value []byte) ([]byte, error) {
	*d.printed = append(*d.printed, byteutil.ToUint64(byteutil.Padding64Bits(value)))
	return value, nil
}

// runProgram compiles every file, lays them end to end, and runs each as its own range.
func runProgram(t *testing.T, files ...file) ([]uint64, error) {
	t.Helper()

	printed := make([]uint64, 0)
	ev := New(NewEvaluatorOptions{PrintDecimal: decimals{&printed}})

	blocks := make([]ir.Block, 0)
	ranges := make([]file, 0, len(files))
	tops := make([]ir.BlockID, 0, len(files))

	for _, each := range files {
		tokens, err := lexer.New().GetFilledTokens([]byte(each.source))
		if err != nil {
			t.Fatalf("lexer on %q: %v", each.id, err)
		}
		tree, err := parser.New().Parse(parser.ParseInput{
			Filename: each.id + ".ar",
			Tokens:   tokens,
			Module:   each.id,
		})
		if err != nil {
			t.Fatalf("parser on %q: %v", each.id, err)
		}
		compiled, err := emitter.New(emitter.NewEmitterOptions{}).Emit(tree)
		if err != nil {
			t.Fatalf("emitter on %q: %v", each.id, err)
		}

		// Each file numbers its blocks from zero, so every one after the first moves, and the
		// file before it carries on into this one.
		top := ir.BlockID(len(blocks))
		if len(tops) > 0 {
			blocks = ir.GoesOnTo(blocks, tops[len(tops)-1], top)
		}
		blocks = append(blocks, ir.Shifted(compiled, top)...)
		ranges = append(ranges, each)
		tops = append(tops, top)
	}

	for i, each := range ranges {
		var until func(ir.Point) bool
		if i+1 < len(tops) {
			next := ir.Point{Block: tops[i+1]}
			until = func(point ir.Point) bool { return point == next }
		}
		if _, err := ev.EvaluateBlocks(blocks, ir.Point{Block: tops[i]}, until, each.id); err != nil {
			return printed, err
		}
	}
	return printed, nil
}

// Two modules binding the same word are two different names, and a scope from one of them
// reads its own.
//
// This is the example the design was argued on. The module's add sums what it was fed and the
// base its own file bound, which is 10; the entry feeds it the base it bound itself, which is
// 3. Answering 7 would mean the imported scope had read the caller's name instead of its own.
func TestAScopeReadsTheModuleItWasWrittenIn(t *testing.T) {
	printed, err := runProgram(t,
		file{"a/b/c", "ident base = 10;\nident add = defer { feed(0) + feed(1) + base; };"},
		file{"", "use a/b/c as x;\nident base = 3;\nprintd x.add(base, 1);"},
	)
	if err != nil {
		t.Fatalf("running: %v", err)
	}
	if len(printed) != 1 || printed[0] != 14 {
		t.Errorf("printed %v, want [14] — 7 would mean it read the caller's base", printed)
	}
}

// A deferred scope is an index counted in the environ that created it, so two modules both
// having a first one is the ordinary case and not a collision.
func TestEachModuleCountsItsOwnDeferredScopes(t *testing.T) {
	printed, err := runProgram(t,
		file{"one", "ident f = defer { 1; };"},
		file{"two", "ident g = defer { 2; };"},
		file{"", "use one as a;\nuse two as b;\nprintd a.f();\nprintd b.g();"},
	)
	if err != nil {
		t.Fatalf("running: %v", err)
	}
	if len(printed) != 2 || printed[0] != 1 || printed[1] != 2 {
		t.Errorf("printed %v, want [1 2] — both modules keep a scope at index 0", printed)
	}
}

// A call crosses two modules: the entry reaches one, which reaches another.
func TestAModuleCallsAnother(t *testing.T) {
	printed, err := runProgram(t,
		file{"two", "ident double = defer { feed(0) * 2; };"},
		file{"one", "use two as t;\nident quad = defer { t.double(t.double(feed(0))); };"},
		file{"", "use one as o;\nprintd o.quad(3);"},
	)
	if err != nil {
		t.Fatalf("running: %v", err)
	}
	if len(printed) != 1 || printed[0] != 12 {
		t.Errorf("printed %v, want [12]", printed)
	}
}

// And what a module cannot do: read a name from whoever called it. A scope sees the chain of
// its caller, which is how a deferred scope has always worked, but the names it asks for are
// its own module's — so the entry's n is not what it finds, and there is nothing else to find.
func TestAModuleDoesNotReadTheCallersNames(t *testing.T) {
	_, err := runProgram(t,
		file{"m", "ident f = defer { n; };"},
		file{"", "use m as x;\nident n = 5;\nprintd x.f();"},
	)
	if err == nil {
		t.Fatal("expected the name not to be found")
	}
	if !strings.Contains(err.Error(), "m.n not found") {
		t.Errorf("error = %q, want it to say m.n was not found", err)
	}
}

// A program of one file runs where it always ran, and is not a module of anything.
func TestAProgramOfOneFileIsUnchanged(t *testing.T) {
	printed, err := runProgram(t, file{"", "ident a = 1;\nident f = defer { a + 1; };\nprintd f();"})
	if err != nil {
		t.Fatalf("running: %v", err)
	}
	if len(printed) != 1 || printed[0] != 2 {
		t.Errorf("printed %v, want [2]", printed)
	}
}
