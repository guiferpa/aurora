package evaluator

import (
	"bytes"
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
	"github.com/guiferpa/aurora/wire/ir"
)

// runWithTapeSize compiles and evaluates source with the given tape size, returning the
// value of the last expression.
func runWithTapeSize(t *testing.T, source string, tapeSize int) []byte {
	t.Helper()

	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	tree, err := parser.New().Parse(parser.ParseInput{Filename: "main.ar", Tokens: tokens, TapeSize: tapeSize})
	if err != nil {
		t.Fatalf("parser: %v", err)
	}
	insts, err := emitter.New(emitter.NewEmitterOptions{TapeSize: tapeSize}).Emit(tree)
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}
	returns, err := New(NewEvaluatorOptions{TapeSize: tapeSize}).EvaluateBlocks(insts, ir.Point{}, nil, "")
	if err != nil {
		t.Fatalf("evaluator: %v", err)
	}

	labels := slices.Collect(maps.Keys(returns))
	slices.SortFunc(labels, func(a, b string) int {
		ia, _ := strconv.ParseInt(a, 10, 64)
		ib, _ := strconv.ParseInt(b, 10, 64)
		return int(ia - ib)
	})
	if len(labels) == 0 {
		t.Fatal("no value produced")
	}
	return returns[labels[len(labels)-1]]
}

// runAndError compiles and runs source, returning the value of the last expression and any
// error, for the cases where the error is the point.
func runAndError(t *testing.T, source string, tapeSize int) ([]byte, error) {
	t.Helper()

	tokens, err := lexer.New().GetFilledTokens([]byte(source))
	if err != nil {
		return nil, err
	}
	tree, err := parser.New().Parse(parser.ParseInput{Filename: "main.ar", Tokens: tokens, TapeSize: tapeSize})
	if err != nil {
		return nil, err
	}
	insts, err := emitter.New(emitter.NewEmitterOptions{TapeSize: tapeSize}).Emit(tree)
	if err != nil {
		return nil, err
	}
	returns, err := New(NewEvaluatorOptions{TapeSize: tapeSize}).EvaluateBlocks(insts, ir.Point{}, nil, "")
	if err != nil {
		return nil, err
	}

	labels := slices.Collect(maps.Keys(returns))
	slices.SortFunc(labels, func(a, b string) int {
		ia, _ := strconv.ParseInt(a, 10, 64)
		ib, _ := strconv.ParseInt(b, 10, 64)
		return int(ia - ib)
	})
	if len(labels) == 0 {
		return nil, nil
	}
	return returns[labels[len(labels)-1]], nil
}

// Every value is a tape of the configured width — numbers and conditions alike.
func TestValuesHaveTheTapeWidth(t *testing.T) {
	sources := []string{"1 + 1;", "true;", "false;", "1 equals 1;", "2 bigger 1;"}
	for _, size := range []int{1, 2, 8, 32} {
		for _, source := range sources {
			t.Run(strconv.Itoa(size)+"/"+source, func(t *testing.T) {
				if got := runWithTapeSize(t, source, size); len(got) != size {
					t.Errorf("%q produced %d bytes, want %d: %v", source, len(got), size, got)
				}
			})
		}
	}
}

// Arithmetic wraps at the tape width, not at 64 bits.
func TestArithmeticWrapsAtTapeWidth(t *testing.T) {
	cases := []struct {
		name   string
		source string
		size   int
		want   uint64
	}{
		{name: "one byte wraps at 256", source: "255 + 1;", size: 1, want: 0},
		{name: "one byte wraps past 256", source: "255 + 2;", size: 1, want: 1},
		{name: "one byte holds 255", source: "254 + 1;", size: 1, want: 255},
		{name: "two bytes wrap at 65536", source: "65535 + 1;", size: 2, want: 0},
		{name: "two bytes hold 65535", source: "65534 + 1;", size: 2, want: 65535},
		{name: "eight bytes keep working", source: "4294967295 + 1;", size: 8, want: 4294967296},
		{name: "multiplication wraps too", source: "16 * 16;", size: 1, want: 0},
		{name: "subtraction borrows within the tape", source: "0 - 1;", size: 1, want: 255},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runWithTapeSize(t, tc.source, tc.size)
			want := byteutil.PaddingTape(byteutil.FromUint64(tc.want), tc.size)
			if !bytes.Equal(got, want) {
				t.Errorf("%q with %d-byte tapes = %v, want %v", tc.source, tc.size, got, want)
			}
		})
	}
}

// Exponentiation used to go through math.Pow on float64 and lost precision past 2^53.
func TestExponentialIsExact(t *testing.T) {
	got := runWithTapeSize(t, "2 ^ 60;", 8)
	want := byteutil.FromUint64(1 << 60)
	if !bytes.Equal(got, want) {
		t.Errorf("2^60 = %v, want %v", got, want)
	}
}

// A condition is a tape holding 1 or 0: true is indistinguishable from the number 1.
func TestConditionsAreTapesHoldingOneOrZero(t *testing.T) {
	for _, size := range []int{1, 8} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			truth := runWithTapeSize(t, "1 equals 1;", size)
			if !bytes.Equal(truth, byteutil.TrueTape(size)) {
				t.Errorf("true = %v, want %v", truth, byteutil.TrueTape(size))
			}
			if !bytes.Equal(truth, runWithTapeSize(t, "1;", size)) {
				t.Error("true and the number 1 must be the same bytes")
			}
			lie := runWithTapeSize(t, "1 equals 2;", size)
			if !bytes.Equal(lie, byteutil.FalseTape(size)) {
				t.Errorf("false = %v, want %v", lie, byteutil.FalseTape(size))
			}
		})
	}
}

// Booleans in arithmetic need no padding rule any more: they are already tapes.
func TestBooleansInArithmetic(t *testing.T) {
	for _, size := range []int{1, 8} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			got := runWithTapeSize(t, "true + 1;", size)
			want := byteutil.PaddingTape([]byte{2}, size)
			if !bytes.Equal(got, want) {
				t.Errorf("true + 1 = %v, want %v", got, want)
			}
		})
	}
}

// Text is a tape holding its bytes, so it is one tape at every width, and how much text
// fits is how wide the tape is.
func TestTextIsATapeAtEveryWidth(t *testing.T) {
	for _, size := range []int{2, 8, 32} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			got := runWithTapeSize(t, `"hi";`, size)
			if len(got) != size {
				t.Errorf(`"hi" produced %d bytes, want %d (one tape)`, len(got), size)
			}
			if want := byteutil.PaddingTape([]byte("hi"), size); !bytes.Equal(got, want) {
				t.Errorf(`"hi" = %v, want %v`, got, want)
			}

			// Text of one character is the tape its number is, so arithmetic needs no rule
			// of its own: 1 + "a" is 98.
			sum := runWithTapeSize(t, `1 + "a";`, size)
			if want := byteutil.PaddingTape([]byte{98}, size); !bytes.Equal(sum, want) {
				t.Errorf(`1 + "a" = %v, want %v`, sum, want)
			}
		})
	}
}
