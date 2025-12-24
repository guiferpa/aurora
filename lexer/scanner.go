package lexer

var keywordTags = []Tag{
	TagIdent,
	TagIf,
	TagElse,
	TagBranch,
	TagCallPrint,
	TagEcho,
	TagTrue,
	TagFalse,
	TagEquals,
	TagDifferent,
	TagBigger,
	TagSmaller,
	TagOr,
	TagAnd,
	TagHead,
	TagTail,
	TagPush,
	TagPull,
	TagArguments,
	TagAssert,
}

var Keywords = func() map[string]Tag {
	m := make(map[string]Tag, len(keywordTags))
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

func scanOneChar(c byte) (Tag, bool) {
	switch c {
	case '(':
		return TagOParen, true
	case ')':
		return TagCParen, true
	case '{':
		return TagOCurBrk, true
	case '}':
		return TagCCurBrk, true
	case '[':
		return TagOBrk, true
	case ']':
		return TagCBrk, true
	case ';':
		return TagSemicolon, true
	case ':':
		return TagColon, true
	case ',':
		return TagComma, true
	case '=':
		return TagAssign, true
	case '+':
		return TagSum, true
	case '-':
		return TagSub, true
	case '*':
		return TagMult, true
	case '/':
		return TagDiv, true
	case '^':
		return TagExpo, true
	default:
		return Tag{}, false
	}
}

func scanTwoChars(bs []byte) (bool, Tag, []byte) {
	if len(bs) < 2 {
		return false, Tag{}, nil
	}

	if bs[0] == '#' && bs[1] == '-' {
		return true, TagComment, bs[:2]
	}

	return false, Tag{}, nil
}

func scanWord(bs []byte) (bool, Tag, []byte) {
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
				return false, Tag{}, nil
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

func scanIdentifier(bs []byte) (bool, Tag, []byte) {
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
				return false, Tag{}, nil
			}
		}

		break
	}

	if i == 0 {
		return false, Tag{}, nil
	}

	return true, TagId, bs[:i]
}

func scanNumber(bs []byte) (bool, Tag, []byte) {
	if len(bs) == 0 || !isDigit(bs[0]) {
		return false, Tag{}, nil
	}

	i := 0

	if len(bs) >= 2 && bs[0] == '0' && (bs[1] == 'x' || bs[1] == 'X') {
		i = 2
		for i < len(bs) && isHexDigit(bs[i]) {
			i++
		}
		return true, TagNumber, bs[:i]
	}

	for i < len(bs) && (isDigit(bs[i]) || bs[i] == '_') {
		i++
	}

	return true, TagNumber, bs[:i]
}

func scanString(bs []byte) (bool, Tag, []byte) {
	if len(bs) == 0 || !isQuote(bs[0]) {
		return false, Tag{}, nil
	}

	i := 1
	for i < len(bs) && !isQuote(bs[i]) {
		i++
	}

	if i < len(bs) && isQuote(bs[i]) {
		i++
	}

	return true, TagString, bs[:i]
}

func scanWhitespace(bs []byte) (bool, Tag, []byte) {
	i := 0
	for i < len(bs) && isSpace(bs[i]) {
		i++
	}

	if i == 0 {
		return false, Tag{}, nil
	}

	return true, TagWhitespace, bs[:i]
}

func ScanToken(bs []byte) (bool, Tag, []byte) {
	if len(bs) == 0 {
		return false, Tag{}, nil
	}

	c := bs[0]

	if tag, ok := scanOneChar(c); ok {
		return true, tag, bs[:1]
	}

	if matched, tag, match := scanTwoChars(bs); matched {
		return true, tag, match
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
		return true, TagBreakLine, bs[:1]
	}

	return false, Tag{}, nil
}
