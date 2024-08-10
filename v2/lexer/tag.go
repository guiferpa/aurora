package lexer

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
)

const (
	IDENT         = "IDENT"     // idenTag
	ASSIGN        = "ASSIGN"    // =
	O_PAREN       = "O_PAREN"   // (
	C_PAREN       = "C_PAREN"   // )
	EQUALS        = "EQUALS"    // equals
	DIFFERENT     = "DIFFERENT" // differenTag
	BIGGER        = "BIGGER"    // bigger
	SMALLER       = "SMALLER"   // smaller
	SUM           = "SUM"       // +
	SUB           = "SUB"       // -
	COMMENT       = "COMMENT"   // --
	O_BRK         = "O_BRK"     // [
	C_BRK         = "C_BRK"     // ]
	O_CUR_BRK     = "O_CUR_BRK" // {
	C_CUR_BRK     = "C_CUR_BRK" // }
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
	TagBreakLine   = Tag{BREAK_LINE, "", "^[\\r\\n]"}
	TagWhitespace  = Tag{WHITESPACE, " ", "^[ ]+"}
	TagIdent       = Tag{IDENT, "ident", "^ident"}
	TagAssign      = Tag{ASSIGN, "=", "^="}
	TagOParen      = Tag{O_PAREN, "(", "^\\("}
	TagCParen      = Tag{C_PAREN, ")", "^\\)"}
	TagEquals      = Tag{EQUALS, "equals", "^equals"}
	TagDifferent   = Tag{DIFFERENT, "different", "^different"}
	TagBigger      = Tag{BIGGER, "bigger", "^bigger"}
	TagSmaller     = Tag{SMALLER, "smaller", "^smaller"}
	TagSum         = Tag{SUM, "+", "^\\+"}
	TagSub         = Tag{SUB, "-", "^\\-"}
	TagComment     = Tag{COMMENT, "--", "^\\#\\-"}
	TagOBrk        = Tag{O_BRK, "[", "^\\["}
	TagCBrk        = Tag{C_BRK, "]", "^\\]"}
	TagOCurBrk     = Tag{O_CUR_BRK, "{", "^{"}
	TagCCurBrk     = Tag{C_CUR_BRK, "}", "^}"}
	TagComma       = Tag{COMMA, ",", "^,"}
	TagIf          = Tag{IF, "if", "^if"}
	TagColon       = Tag{COLON, ":", "^:"}
	TagSemicolon   = Tag{SEMICOLON, ";", "^;"}
	TagId          = Tag{ID, "", "^[A-Za-z][A-Za-z0-9-_?!><]*"}
	TagNumber      = Tag{NUMBER, "", "^[0-9][0-9_]*\\b"}
	TagEndOfBuffer = Tag{END_OF_BUFFER, "<EOB>", ""}
)

var processableTags = []Tag{
	TagWhitespace,
	TagBreakLine,
	TagComment,
	TagIf,
	TagIdent,
	TagAssign,
	TagOParen,
	TagCParen,
	TagEquals,
	TagDifferent,
	TagBigger,
	TagSmaller,
	TagOBrk,
	TagCBrk,
	TagOCurBrk,
	TagCCurBrk,
	TagComma,
	TagColon,
	TagSemicolon,
	TagId,
	TagSum,
	TagSub,
	TagNumber,
}

func GeTagProcessbleTags() []Tag {
	return processableTags
}

func MatchTagRule(bs []byte) (bool, Tag, []byte) {
	for _, v := range GeTagProcessbleTags() {
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
