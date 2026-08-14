package evaluator

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
)

// A deferred scope is a value like any other: a tape of the configured width. It used to be
// the hex key of its own storage — 16 bytes of ASCII text that ignored the tape size, so
// "ident b = defer {};" showed a row of zeros where "ident a = {};" showed one.
func TestDeferValueIsATape(t *testing.T) {
	for _, size := range []int{1, 2, 8, 32} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			deferred := runWithTapeSize(t, "defer {};", size)
			if len(deferred) != size {
				t.Errorf("a defer produced %d bytes, want %d: %v", len(deferred), size, deferred)
			}

			block := runWithTapeSize(t, "{};", size)
			if len(deferred) != len(block) {
				t.Errorf("a defer and a block produced %d and %d bytes", len(deferred), len(block))
			}
		})
	}
}

// The value is the index of the scope, so the first one is zero — the same tape as false
// and as the number 0.
func TestFirstDeferIsZero(t *testing.T) {
	got := runWithTapeSize(t, "defer {};", byteutil.DefaultTapeSize)
	if want := byteutil.FalseTape(byteutil.DefaultTapeSize); !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSeveralDefersGetDistinctValues(t *testing.T) {
	first := runWithTapeSize(t, "ident a = defer { 1; };\na;", byteutil.DefaultTapeSize)
	second := runWithTapeSize(t, "ident a = defer { 1; };\nident b = defer { 2; };\nb;", byteutil.DefaultTapeSize)

	if bytes.Equal(first, second) {
		t.Errorf("two defers share the value %v", first)
	}
}

// Each one still calls its own scope.
func TestDefersCallTheirOwnScope(t *testing.T) {
	got := runWithTapeSize(t, `ident one = defer { 11; };
ident two = defer { 22; };
one() + two();
`, byteutil.DefaultTapeSize)

	want := byteutil.PaddingTape(byteutil.FromUint64(33), byteutil.DefaultTapeSize)
	if !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCallWorksAcrossTapeSizes(t *testing.T) {
	for _, size := range []int{1, 2, 8, 32} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			got := runWithTapeSize(t, "ident add = defer { feed(0) + feed(1); };\nadd(2, 3);", size)
			want := byteutil.PaddingTape([]byte{5}, size)
			if !bytes.Equal(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestRecursionSurvives(t *testing.T) {
	got := runWithTapeSize(t, `ident fib = defer {
  ident n = feed(0);
  if n smaller 2 { n; } else { fib(n - 1) + fib(n - 2); };
};
fib(10);
`, byteutil.DefaultTapeSize)

	want := byteutil.PaddingTape(byteutil.FromUint64(55), byteutil.DefaultTapeSize)
	if !bytes.Equal(got, want) {
		t.Errorf("fib(10) = %v, want %v", got, want)
	}
}

// Calling a value with no scope behind it is still an error.
func TestCallingSomethingThatIsNotADeferFails(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{name: "a number with no scope behind it", source: "ident a = 7;\na();"},
		{name: "the value of a block", source: "ident a = { 1; };\na();"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runAndError(t, tc.source, byteutil.DefaultTapeSize)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), "not a deferred scope") {
				t.Errorf("error = %q, want it to say the value is not a deferred scope", err)
			}
		})
	}
}

// A reference is a tape holding an index, so a number equal to that index is that
// reference — there is nothing in the bytes to tell them apart. This is the same bargain
// the language already makes for true and 1, or for false and 0: an untyped language
// cannot distinguish values that are the same bytes, and does not pretend to.
func TestANumberEqualToAnIndexIsThatReference(t *testing.T) {
	got := runWithTapeSize(t, "ident d = defer { 42; };\nident a = 0;\na();", byteutil.DefaultTapeSize)

	want := byteutil.PaddingTape([]byte{42}, byteutil.DefaultTapeSize)
	if !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v — a value equal to a scope's index calls that scope", got, want)
	}
}

func TestDeferBlobRoundTrip(t *testing.T) {
	blob := encodeDeferBlob(3, 9, "0a")
	from, to, key, ok := decodeDeferBlob(blob)
	if !ok {
		t.Fatal("a blob we just wrote did not decode")
	}
	if from != 3 || to != 9 || key != "0a" {
		t.Errorf("got (%d, %d, %q), want (3, 9, %q)", from, to, key, "0a")
	}
}

func TestDecodeDeferBlobRejectsAnythingElse(t *testing.T) {
	cases := []struct {
		name string
		blob []byte
	}{
		{name: "empty", blob: nil},
		{name: "too short", blob: []byte{deferMark, 1, 2, 3}},
		{name: "no mark", blob: bytes.Repeat([]byte{0}, 32)},
		{name: "key longer than the blob", blob: append([]byte{deferMark}, append(bytes.Repeat([]byte{0}, 16), 200)...)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, ok := decodeDeferBlob(tc.blob); ok {
				t.Error("expected the blob to be rejected")
			}
		})
	}
}
