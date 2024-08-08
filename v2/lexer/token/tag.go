package token

import (
	"errors"
	"strings"
)

const (
	IDENT     = "IDENT"     // ident
	ASSIGN    = "ASSIGN"    // =
	O_PAREN   = "O_PAREN"   // (
	C_PAREN   = "C_PAREN"   // )
	EQUALS    = "EQUALS"    // equals
	DIFFERENT = "DIFFERENT" // different
	BIGGER    = "BIGGER"    // bigger
	SMALLER   = "SMALLER"   // smaller
	SUM       = "SUM"       // +
	SUB       = "SUB"       // -
	COMMENT   = "COMMENT"   // --
	O_BRK     = "O_BRK"     // {
	C_BRK     = "C_BRK"     // }
	COMMA     = "COMMA"     // ,
	IF        = "IF"        // if
	COLON     = "COLON"     // :
	SEMICOLON = "SEMICOLON" // ;
	ID        = "ID"
)

type Tag struct {
	Id      string
	Keyword string
	Rule    string
}

var (
	tIdent     = Tag{IDENT, "ident", "^ident"}
	tAssign    = Tag{ASSIGN, "=", "^="}
	tOParen    = Tag{O_PAREN, "(", "^\\("}
	tCParen    = Tag{C_PAREN, ")", "^\\)"}
	tEquals    = Tag{EQUALS, "equals", "^equals"}
	tDifferent = Tag{DIFFERENT, "different", "^different"}
	tBigger    = Tag{BIGGER, "bigger", "^bigger"}
	tSmaller   = Tag{SMALLER, "smaller", "^smaller"}
	tSum       = Tag{SUM, "+", "\\+"}
	tSub       = Tag{SUB, "-", "\\-"}
	tComment   = Tag{COMMENT, "--", "^\\#\\-"}
	tOBrk      = Tag{O_BRK, "{", "^{"}
	tCBrk      = Tag{C_BRK, "}", "^}"}
	tComma     = Tag{COMMA, ",", "^,"}
	tIf        = Tag{IF, "if", "^if"}
	tColon     = Tag{COLON, ":", "^:"}
	tSemicolon = Tag{SEMICOLON, ";", "^;"}
	tId        = Tag{ID, "", "^[A-Za-z][A-Za-z0-9-_?!><]*"}
)

var processableTags = []Tag{
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
}

func GetProcessbleTags() []Tag {
	return processableTags
}

func GetTag(c string) (Tag, error) {
	for _, v := range processableTags {
		if strings.Compare(v.Id, c) == 0 {
			return v, nil
		}
	}
	return Tag{}, errors.New("Tag not found")
}
