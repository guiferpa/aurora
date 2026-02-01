package lexer

import (
	"testing"
)

// 1. Current Implementation (Baseline)

func scanWordCurrent(bs []byte) (bool, Tag, []byte) {
	i := 0
	for i < len(bs) {
		c := bs[i]

		if isIdentChar(c) {
			i++
			continue
		}

		// Explicit check to STOP on '=' if preceded by specific symbols
		// This enforces that 'my>=var' is NOT a single identifier in the current language
		if c == '=' && i > 0 {
			prevChar := bs[i-1]
			if prevChar == '>' || prevChar == '<' || prevChar == '!' {
				return false, Tag{}, nil
			}
		}

		break
	}

	if i == 0 {
		return false, Tag{}, nil
	}

	// Keyword lookup included for fair comparison if we want end-to-end word scanning speed
	if tag, isKeyword := Keywords[string(bs[:i])]; isKeyword {
		return true, tag, bs[:i]
	}

	return true, TagId, bs[:i]
}

// 2. Strict Implementation (Alphanumeric Only)

func isIdentCharStrict(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func scanWordStrict(bs []byte) (bool, Tag, []byte) {
	i := 0
	for i < len(bs) {
		c := bs[i]
		if isIdentCharStrict(c) {
			i++
			continue
		}
		break
	}

	if i == 0 {
		return false, Tag{}, nil
	}

	if tag, isKeyword := Keywords[string(bs[:i])]; isKeyword {
		return true, tag, bs[:i]
	}

	return true, TagId, bs[:i]
}

// 3. Max Flexibility Implementation

func scanWordMaxFlex(bs []byte) (bool, Tag, []byte) {
	i := 0
	for i < len(bs) {
		c := bs[i]

		if isIdentChar(c) {
			i++
			continue
		}

		// Allow '=' if it is part of a symbol sequence
		// Logic: If we hit '=', and the previous char was a symbol OR '=', we consume it.
		// This allows 'my>==var' to be one token.
		// Note: isIdentChar includes '>', '<', '!', '?', '-'
		if c == '=' && i > 0 {
			prevChar := bs[i-1]
			if prevChar == '>' || prevChar == '<' || prevChar == '!' || prevChar == '=' || prevChar == '-' {
				i++
				continue
			}
		}

		break
	}

	if i == 0 {
		return false, Tag{}, nil
	}

	if tag, isKeyword := Keywords[string(bs[:i])]; isKeyword {
		return true, tag, bs[:i]
	}

	return true, TagId, bs[:i]
}

// Benchmarks

var (
	inputStandard = []byte("myVariable_123")

	// Complex cases
	inputComplex1 = []byte("my>=var")
	inputComplex2 = []byte("my>=====var")
)

// -- Standard ID Benchmarks (The "Tax" Test) --

func Benchmark_ScanWord_Current_Standard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		scanWordCurrent(inputStandard)
	}
}

func Benchmark_ScanWord_Strict_Standard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		scanWordStrict(inputStandard)
	}
}

func Benchmark_ScanWord_MaxFlex_Standard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		scanWordMaxFlex(inputStandard)
	}
}

// -- Complex Case Benchmarks (Behavior & Perf) --

func Benchmark_ScanWord_Current_Complex_GTE(b *testing.B) {
	for i := 0; i < b.N; i++ {
		scanWordCurrent(inputComplex1)
	}
}

func Benchmark_ScanWord_MaxFlex_Complex_GTE(b *testing.B) {
	for i := 0; i < b.N; i++ {
		scanWordMaxFlex(inputComplex1)
	}
}

func Benchmark_ScanWord_MaxFlex_Complex_LongArrows(b *testing.B) {
	for i := 0; i < b.N; i++ {
		scanWordMaxFlex(inputComplex2)
	}
}

// -- Verification Tests to prove behavior --

func TestScannerVariants_Behavior(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		fn          func([]byte) (bool, Tag, []byte)
		wantMatch   string
		wantSuccess bool
		desc        string
	}{
		// Current
		{"Current_Standard", "myVar", scanWordCurrent, "myVar", true, "Standard ID"},
		{"Current_GTE", "my>=var", scanWordCurrent, "", false, "Should fail explicitly on >="},
		{"Current_Arrow", "my->var", scanWordCurrent, "my->var", true, "Should allow ->"},

		// Strict
		{"Strict_Standard", "myVar", scanWordStrict, "myVar", true, "Standard ID"},
		{"Strict_Underscore", "my_var", scanWordStrict, "my_var", true, "Standard ID"},
		{"Strict_Arrow", "my->var", scanWordStrict, "my", true, "Should stop at -"},

		// MaxFlex
		{"MaxFlex_Standard", "myVar", scanWordMaxFlex, "myVar", true, "Standard ID"},
		{"MaxFlex_GTE", "my>=var", scanWordMaxFlex, "my>=var", true, "Should consume full ID including >="},
		{"MaxFlex_GiantArrow", "my>=====var", scanWordMaxFlex, "my>=====var", true, "Should consume arbitrary length symbols"},
		{"MaxFlex_Arrow", "my->var", scanWordMaxFlex, "my->var", true, "Should allow ->"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, _, match := tt.fn([]byte(tt.input))

			if matched != tt.wantSuccess {
				t.Errorf("%s: success = %v, want %v", tt.desc, matched, tt.wantSuccess)
				return
			}

			if tt.wantSuccess && string(match) != tt.wantMatch {
				t.Errorf("%s: got %q, want %q", tt.desc, string(match), tt.wantMatch)
			}
		})
	}
}
