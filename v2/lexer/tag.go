package lexer

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
)

const (
	IDENT         = "IDENT"     // ident
	ASSIGN        = "ASSIGN"    // =
	O_PAREN       = "O_PAREN"   // (
	C_PAREN       = "C_PAREN"   // )
	EQUALS        = "EQUALS"    // equals
	DIFFERENT     = "DIFFERENT" // different
	BIGGER        = "BIGGER"    // bigger
	SMALLER       = "SMALLER"   // smaller
	SUM           = "SUM"       // +
	SUB           = "SUB"       // -
	COMMENT       = "COMMENT"   // --
	O_BRK         = "O_BRK"     // {
	C_BRK         = "C_BRK"     // }
	COMMA         = "COMMA"     // ,
	IF            = "IF"        // if
	COLON         = "COLON"     // :
	SEMICOLON     = "SEMICOLON" // ;
	ID            = "ID"
	NUMBER        = "NUMBER"
	WHITESPACE    = "WHITESPACE"
	BREAK_LINE    = "BREAK_LINE"
	END_OF_BUFFER = "END_OF_BUFFER"
)

type Tag struct {
	Id      string
	Keyword string
	Rule    string
}

var (
	tBreakLine   = Tag{BREAK_LINE, "", "^[\\r\\n]"}
	tWhitespace  = Tag{WHITESPACE, " ", "^[ ]+"}
	tIdent       = Tag{IDENT, "ident", "^ident"}
	tAssign      = Tag{ASSIGN, "=", "^="}
	tOParen      = Tag{O_PAREN, "(", "^\\("}
	tCParen      = Tag{C_PAREN, ")", "^\\)"}
	tEquals      = Tag{EQUALS, "equals", "^equals"}
	tDifferent   = Tag{DIFFERENT, "different", "^different"}
	tBigger      = Tag{BIGGER, "bigger", "^bigger"}
	tSmaller     = Tag{SMALLER, "smaller", "^smaller"}
	tSum         = Tag{SUM, "+", "^\\+"}
	tSub         = Tag{SUB, "-", "^\\-"}
	tComment     = Tag{COMMENT, "--", "^\\#\\-"}
	tOBrk        = Tag{O_BRK, "{", "^{"}
	tCBrk        = Tag{C_BRK, "}", "^}"}
	tComma       = Tag{COMMA, ",", "^,"}
	tIf          = Tag{IF, "if", "^if"}
	tColon       = Tag{COLON, ":", "^:"}
	tSemicolon   = Tag{SEMICOLON, ";", "^;"}
	tId          = Tag{ID, "", "^[A-Za-z][A-Za-z0-9-_?!><]*"}
	tNumber      = Tag{NUMBER, "", "^[0-9][0-9_]*\\b"}
	tEndOfBuffer = Tag{END_OF_BUFFER, "<EOB>", ""}
)

var processableTags = []Tag{
	tWhitespace,
	tBreakLine,
	tComment,
	tIf,
	tIdent,
	tAssign,
	tOParen,
	tCParen,
	tEquals,
	tDifferent,
	tBigger,
	tSmaller,
	tOBrk,
	tCBrk,
	tComma,
	tColon,
	tSemicolon,
	tId,
	tSum,
	tSub,
	tNumber,
}

func GetProcessbleTags() []Tag {
	return processableTags
}

func MatchTagRuleGivenBytes(bs []byte) (bool, Tag, []byte) {
	for _, v := range GetProcessbleTags() {
		re := regexp.MustCompile(v.Rule)
		match := re.FindString(string(bs))
		if len(match) > 0 {
			return true, v, bytes.NewBufferString(match).Bytes()
		}
	}
	return false, Tag{}, []byte{}
}

func GetTag(c string) (Tag, error) {
	for _, v := range processableTags {
		if strings.Compare(v.Id, c) == 0 {
			return v, nil
		}
	}
	return Tag{}, errors.New("no tag found")
}
