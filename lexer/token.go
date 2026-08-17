package lexer

import (
	"fmt"

	"github.com/guiferpa/aurora/wire/token"
)

func (l *Lexer) GetTokens(bs []byte) ([]token.Token, error) {
	cursor := 0
	col := cursor + 1
	line := 1
	length := len(bs)
	tokens := make([]token.Token, 0)
	isComment := false
	for cursor < length {
		matched, tag, match := MatchToken(bs[cursor:])
		if !matched && !isComment {
			return tokens, &token.Error{
				Message: fmt.Sprintf("unexpected character at line %d, column %d", line, col),
				Line:    line,
				Column:  col,
				Offset:  cursor,
				Length:  1,
			}
		}
		if !isComment {
			tokens = append(tokens, token.New(match, tag, line, col, cursor))
		}
		if len(match) == 0 {
			cursor++
		}
		cursor = cursor + len(match)

		if tag.Id == token.COMMENT_LINE {
			isComment = true
		}

		if tag.Id == token.BREAK_LINE {
			isComment = false
			line++
			col = 1
		} else {
			col = col + len(match)
		}
	}
	return append(tokens, token.New([]byte{}, token.TagEOF, line, col, cursor)), nil
}

func (l *Lexer) GetFilledTokens(bs []byte) ([]token.Token, error) {
	toks, err := l.GetTokens(bs)
	if err != nil {
		return toks, err
	}
	ntoks := make([]token.Token, 0)
	for _, tok := range toks {
		if tok.GetTag().Id == token.WHITESPACE || tok.GetTag().Id == token.BREAK_LINE || tok.GetTag().Id == token.COMMENT_LINE {
			continue
		}
		ntoks = append(ntoks, tok)
	}
	return ntoks, nil
}
