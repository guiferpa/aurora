package lexer

import "github.com/guiferpa/aurora/wire/token"

func MatchToken(bs []byte) (bool, token.Tag, []byte) {
	return ScanToken(bs)
}
