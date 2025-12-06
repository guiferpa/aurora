package lexer

import (
	"testing"
)

var (
	// Simple expression
	simpleInput = []byte(`ident foo = 123;`)

	// Medium complexity - if statement
	mediumInput = []byte(`ident result = if 10 bigger 5 { 100; } else { 200; };`)

	// Complex - multiple statements with function calls
	complexInput = []byte(`
ident fib = {
  ident n = arguments 0;
  if n smaller 1 or n equals 1 { n; } else { fib(n - 1) + fib(n - 2); };
};
ident result = fib(10);
print result;
`)

	// Hexadecimal numbers
	hexInput = []byte(`ident hex_ff = 0xFF; ident hex_10 = 0x10; ident sum = hex_ff + hex_10;`)

	// String heavy
	stringInput = []byte(`ident greeting = "hello"; ident name = "world"; print greeting; print name;`)

	// Many tokens
	manyTokensInput = []byte(`ident a = 1; ident b = 2; ident c = 3; ident d = 4; ident e = 5; ident f = 6; ident g = 7; ident h = 8; ident i = 9; ident j = 10;`)
)

func BenchmarkScanToken_Keyword(b *testing.B) {
	input := []byte(`ident`)
	for i := 0; i < b.N; i++ {
		ScanToken(input)
	}
}

func BenchmarkScanToken_Identifier(b *testing.B) {
	input := []byte(`myVariable123`)
	for i := 0; i < b.N; i++ {
		ScanToken(input)
	}
}

func BenchmarkScanToken_Number(b *testing.B) {
	input := []byte(`123456`)
	for i := 0; i < b.N; i++ {
		ScanToken(input)
	}
}

func BenchmarkScanToken_HexNumber(b *testing.B) {
	input := []byte(`0xABCDEF`)
	for i := 0; i < b.N; i++ {
		ScanToken(input)
	}
}

func BenchmarkScanToken_String(b *testing.B) {
	input := []byte(`"hello world"`)
	for i := 0; i < b.N; i++ {
		ScanToken(input)
	}
}

func BenchmarkScanToken_SingleChar(b *testing.B) {
	input := []byte(`(`)
	for i := 0; i < b.N; i++ {
		ScanToken(input)
	}
}

func BenchmarkScanToken_Comment(b *testing.B) {
	input := []byte(`#- this is a comment`)
	for i := 0; i < b.N; i++ {
		ScanToken(input)
	}
}

func BenchmarkGetTokens_Simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetTokens(simpleInput)
	}
}

func BenchmarkGetTokens_Medium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetTokens(mediumInput)
	}
}

func BenchmarkGetTokens_Complex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetTokens(complexInput)
	}
}

func BenchmarkGetTokens_Hex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetTokens(hexInput)
	}
}

func BenchmarkGetTokens_Strings(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetTokens(stringInput)
	}
}

func BenchmarkGetTokens_ManyTokens(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetTokens(manyTokensInput)
	}
}

func BenchmarkGetFilledTokens_Complex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetFilledTokens(complexInput)
	}
}
func BenchmarkIsIdentChar(b *testing.B) {
	chars := []byte("abcABC123_")
	for i := 0; i < b.N; i++ {
		for _, c := range chars {
			isIdentChar(c)
		}
	}
}

func BenchmarkKeywordsLookup(b *testing.B) {
	words := []string{"if", "else", "ident", "print", "foo", "bar", "myVar"}
	for i := 0; i < b.N; i++ {
		for _, w := range words {
			_ = Keywords[w]
		}
	}
}
