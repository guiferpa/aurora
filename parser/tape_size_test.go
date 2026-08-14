package parser

import (
	"errors"
	"strings"
	"testing"

	"github.com/guiferpa/aurora/byteutil"
	"github.com/guiferpa/aurora/lexer"
)

func parseWithTapeSize(t *testing.T, source string, tapeSize int) (AST, error) {
	t.Helper()
	tokens, err := lexer.New(lexer.NewLexerOptions{}).GetFilledTokens([]byte(source))
	if err != nil {
		t.Fatalf("lexer: %v", err)
	}
	return New(NewParserOptions{
		Filename: "main.ar",
		Tokens:   tokens,
		TapeSize: tapeSize,
	}).Parse()
}

// A literal that the tape cannot hold is a compile-time error, not a silent truncation.
func TestLiteralMustFitInTape(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		size    int
		wantErr string
	}{
		{name: "300 does not fit in one byte", source: "ident a = 300;", size: 1, wantErr: "does not fit in a 1-byte tape"},
		{name: "256 does not fit in one byte", source: "ident a = 256;", size: 1, wantErr: "does not fit"},
		{name: "hexadecimal is checked too", source: "ident a = 0x100;", size: 1, wantErr: "does not fit"},
		{name: "70000 does not fit in two bytes", source: "ident a = 70_000;", size: 2, wantErr: "does not fit in a 2-byte tape"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseWithTapeSize(t, tc.source, tc.size)
			if err == nil {
				t.Fatal("expected a compile-time error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to mention %q", err, tc.wantErr)
			}
			// The language server underlines it, so the position has to travel with the error.
			var perr *lexer.Error
			if !errors.As(err, &perr) {
				t.Fatalf("error is not positioned: %T", err)
			}
			if perr.Line != 1 || perr.Offset == 0 {
				t.Errorf("position = line %d offset %d, want line 1 and a non-zero offset", perr.Line, perr.Offset)
			}
		})
	}
}

func TestLiteralThatFitsIsAccepted(t *testing.T) {
	cases := []struct {
		source string
		size   int
	}{
		{source: "ident a = 255;", size: 1},
		{source: "ident a = 65535;", size: 2},
		{source: "ident a = 300;", size: 2},
		{source: "ident a = 4294967295;", size: 8},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			if _, err := parseWithTapeSize(t, tc.source, tc.size); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// A tape literal cannot carry more values than the tape has bytes.
func TestTapeLiteralMustFitInTape(t *testing.T) {
	if _, err := parseWithTapeSize(t, "ident a = [1, 2, 3];", 2); err == nil {
		t.Error("expected an error for three values in a 2-byte tape")
	} else if !strings.Contains(err.Error(), "tape holds 2 bytes") {
		t.Errorf("error = %q, want it to mention the tape width", err)
	}

	if _, err := parseWithTapeSize(t, "ident a = [1, 2];", 2); err != nil {
		t.Errorf("two values fit in a 2-byte tape: %v", err)
	}
}

// Boolean literals are tapes of the configured width.
func TestBooleanLiteralWidth(t *testing.T) {
	for _, size := range []int{1, 8, 32} {
		ns, err := parseWithTapeSize(t, "true;", size)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		literal, ok := ns.Nodes[0].(BooleanLiteral)
		if !ok {
			t.Fatalf("unexpected node: %T", ns.Nodes[0])
		}
		if len(literal.Value) != size {
			t.Errorf("true has %d bytes, want %d", len(literal.Value), size)
		}
		if literal.Value[len(literal.Value)-1] != 1 {
			t.Errorf("true = %v, want a tape holding 1", literal.Value)
		}
	}
}

func TestDefaultTapeSizeIsEightBytes(t *testing.T) {
	ns, err := parseWithTapeSize(t, "true;", 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := len(ns.Nodes[0].(BooleanLiteral).Value); got != byteutil.DefaultTapeSize {
		t.Errorf("unset tape size produced %d bytes, want %d", got, byteutil.DefaultTapeSize)
	}
}
