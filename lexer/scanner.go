package lexer

import "github.com/guiferpa/aurora/wire/token"

var keywordTags = []token.Tag{
	token.TagIdent,
	token.TagIf,
	token.TagElse,
	token.TagBranch,
	token.TagDefer,
	token.TagPrintBytes,
	token.TagPrintChars,
	token.TagPrintDec,
	token.TagTrue,
	token.TagFalse,
	token.TagEquals,
	token.TagDifferent,
	token.TagBigger,
	token.TagSmaller,
	token.TagOr,
	token.TagAnd,
	token.TagHead,
	token.TagTail,
	token.TagPush,
	token.TagPull,
	token.TagFeed,
	token.TagAssert,
	token.TagShape,
	token.TagAs,
	token.TagUse,
	token.TagReturns,
}

var Keywords = func() map[string]token.Tag {
	m := make(map[string]token.Tag, len(keywordTags))
	for _, t := range keywordTags {
		m[t.Keyword] = t
	}
	return m
}()

func isLowercaseLetter(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func isUppercaseLetter(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isSpace(c byte) bool {
	return c == ' '
}

func isQuote(c byte) bool {
	return c == '"'
}

func isNewline(c byte) bool {
	return c == '\n' || c == '\r'
}

func isIdentChar(c byte) bool {
	return isLowercaseLetter(c) || isUppercaseLetter(c) || isDigit(c) ||
		c == '_' || c == '-' ||
		c == '?' || c == '!' ||
		c == '>' || c == '<'
}

func scanOneChar(c byte) (token.Tag, bool) {
	switch c {
	case '(':
		return token.TagOParen, true
	case ')':
		return token.TagCParen, true
	case '{':
		return token.TagOCurBrk, true
	case '}':
		return token.TagCCurBrk, true
	case '[':
		return token.TagOBrk, true
	case ']':
		return token.TagCBrk, true
	case ';':
		return token.TagSemicolon, true
	case ':':
		return token.TagColon, true
	case ',':
		return token.TagComma, true
	case '=':
		return token.TagAssign, true
	case '+':
		return token.TagSum, true
	case '-':
		return token.TagSub, true
	case '*':
		return token.TagMult, true
	case '/':
		return token.TagDiv, true
	case '^':
		return token.TagExpo, true
	case '.':
		return token.TagDot, true
	default:
		return token.Tag{}, false
	}
}

func scanTwoChars(bs []byte) (bool, token.Tag, []byte) {
	if len(bs) < 2 {
		return false, token.Tag{}, nil
	}

	if bs[0] == '#' && bs[1] == '-' {
		return true, token.TagComment, bs[:2]
	}

	return false, token.Tag{}, nil
}

func scanWord(bs []byte) (bool, token.Tag, []byte) {
	i := 0
	for i < len(bs) {
		c := bs[i]

		if isIdentChar(c) {
			i++
			continue
		}

		if c == '=' && i > 0 {
			prevChar := bs[i-1]
			if prevChar == '>' || prevChar == '<' || prevChar == '!' {
				return false, token.Tag{}, nil
			}
		}

		break
	}

	if i == 0 {
		return false, token.Tag{}, nil
	}

	if tag, isKeyword := Keywords[string(bs[:i])]; isKeyword {
		return true, tag, bs[:i]
	}

	return true, token.TagId, bs[:i]
}

func scanIdentifier(bs []byte) (bool, token.Tag, []byte) {
	i := 0
	for i < len(bs) {
		c := bs[i]

		if isIdentChar(c) {
			i++
			continue
		}

		if c == '=' && i > 0 {
			prevChar := bs[i-1]
			if prevChar == '>' || prevChar == '<' || prevChar == '!' {
				return false, token.Tag{}, nil
			}
		}

		break
	}

	if i == 0 {
		return false, token.Tag{}, nil
	}

	return true, token.TagId, bs[:i]
}

func scanNumber(bs []byte) (bool, token.Tag, []byte) {
	if len(bs) == 0 || !isDigit(bs[0]) {
		return false, token.Tag{}, nil
	}

	i := 0

	if len(bs) >= 2 && bs[0] == '0' && (bs[1] == 'x' || bs[1] == 'X') {
		i = 2
		for i < len(bs) && isHexDigit(bs[i]) {
			i++
		}
		return true, token.TagNumber, bs[:i]
	}

	for i < len(bs) && (isDigit(bs[i]) || bs[i] == '_') {
		i++
	}

	return true, token.TagNumber, bs[:i]
}

func scanString(bs []byte) (bool, token.Tag, []byte) {
	if len(bs) == 0 || !isQuote(bs[0]) {
		return false, token.Tag{}, nil
	}

	i := 1
	for i < len(bs) && !isQuote(bs[i]) {
		i++
	}

	if i < len(bs) && isQuote(bs[i]) {
		i++
	}

	return true, token.TagString, bs[:i]
}

func scanWhitespace(bs []byte) (bool, token.Tag, []byte) {
	i := 0
	for i < len(bs) && isSpace(bs[i]) {
		i++
	}

	if i == 0 {
		return false, token.Tag{}, nil
	}

	return true, token.TagWhitespace, bs[:i]
}

func ScanToken(bs []byte) (bool, token.Tag, []byte) {
	if len(bs) == 0 {
		return false, token.Tag{}, nil
	}

	c := bs[0]

	if matched, tag, match := scanTwoChars(bs); matched {
		return true, tag, match
	}

	if tag, ok := scanOneChar(c); ok {
		return true, tag, bs[:1]
	}

	if isLowercaseLetter(c) {
		return scanWord(bs)
	}

	if isUppercaseLetter(c) {
		return scanIdentifier(bs)
	}

	if isDigit(c) {
		return scanNumber(bs)
	}

	if isQuote(c) {
		return scanString(bs)
	}

	if isSpace(c) {
		return scanWhitespace(bs)
	}

	if isNewline(c) {
		return true, token.TagBreakLine, bs[:1]
	}

	return false, token.Tag{}, nil
}
