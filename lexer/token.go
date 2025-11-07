package lexer

import "fmt"

type Token interface {
	GetMatch() []byte
	GetTag() Tag
	GetLine() int
	GetColumn() int
	GetCursor() int
}

type tok struct {
	x, y, c int
	tag     Tag
	match   []byte
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

func (t tok) GetCursor() int {
	return t.c
}

func GetTokens(bs []byte) ([]Token, error) {
	cursor := 0
	col := cursor + 1
	line := 1
	length := len(bs)
	tokens := make([]Token, 0)
	isComment := false
	for cursor < length {
		matched, tag, match := MatchTagRule(bs[cursor:])
		if !matched && !isComment {
			return tokens, fmt.Errorf("unexpected character at line %d, column %d", line, col)
		}
		if !isComment {
			tokens = append(tokens, tok{line, col, cursor, tag, match})
		}
		if len(match) == 0 {
			cursor++
		}
		cursor = cursor + len(match)

		if tag.Id == COMMENT_LINE {
			isComment = true
		}

		if tag.Id == BREAK_LINE {
			isComment = false
			line++
			col = 1
		} else {
			col = col + len(match)
		}
	}
	return append(tokens, tok{line, col, cursor, TagEOF, []byte{}}), nil
}

func GetFilledTokens(bs []byte) ([]Token, error) {
	toks, err := GetTokens(bs)
	if err != nil {
		return toks, err
	}
	ntoks := make([]Token, 0)
	for _, tok := range toks {
		if tok.GetTag().Id == WHITESPACE || tok.GetTag().Id == BREAK_LINE || tok.GetTag().Id == COMMENT_LINE {
			continue
		}
		ntoks = append(ntoks, tok)
	}
	return ntoks, nil
}
