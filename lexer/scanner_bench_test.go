package lexer

import (
	"testing"
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
