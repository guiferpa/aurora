package lexer

import (
	"errors"
)

type Token interface {
	GetMatch() []byte
	GetTag() Tag
	GetLine() int
	GetColumn() int
}

type tok struct {
	x, y  int
	tag   Tag
	match []byte
}

func (t tok) GetMatch() []byte {
	return t.match
}

func (t tok) GetTag() Tag {
	return t.tag
}

func (t tok) GetLine() int {
	return t.x
}

func (t tok) GetColumn() int {
	return t.y
}

func GetTokensGivenBytes(bs []byte) ([]Token, error) {
	cursor := 0
	col := cursor + 1
	line := 1
	length := len(bs)
	tokens := make([]Token, 0)
	for cursor < length {
		matched, tag, match := MatchTagRuleGivenBytes(bs[cursor:])
		if !matched {
			return tokens, errors.New("no token matched")
		}
		tokens = append(tokens, tok{line, col, tag, match})
		cursor = cursor + len(match)
		if tag.Id == BREAK_LINE {
			line++
			col = 1
		} else {
			col = col + len(match)
		}
	}
	return append(tokens, tok{line, col, tEndOfBuffer, []byte{}}), nil
}
