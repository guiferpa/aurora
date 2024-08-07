package token

import (
	"regexp"
)

type Token interface {
	GetLine() int
	GetColumn() int
}

type tok struct {
	x, y int
}

func (t tok) GetLine() int {
	return t.x
}

func (t tok) GetColumn() int {
	return t.y
}

func tokenMatchGivenTagRule(bs []byte) (bool, Tag, string) {
	for _, v := range GetProcessbleTags() {
		re := regexp.MustCompile(v.Rule)
		match := re.FindString(string(bs))
		if len(match) > 0 {
			return true, v, match
		}
	}
	return false, Tag{}, ""
}

func GetTokensGivenBytes(bs []byte) []Token {
	tokens := make([]Token, 0)
	return tokens
}

func NewEOFToken(line, column int) Token {
	return tok{line, column}
}
