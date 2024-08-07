package token

import (
	"errors"
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
	tIdent     = Tag{IDENT, "ident", ""}
	tAssign    = Tag{ASSIGN, "=", ""}
	tOParen    = Tag{O_PAREN, "(", ""}
	tCParen    = Tag{C_PAREN, ")", ""}
	tEquals    = Tag{EQUALS, "equals", ""}
	tDifferent = Tag{DIFFERENT, "different", ""}
	tBigger    = Tag{BIGGER, "bigger", ""}
	tSmaller   = Tag{SMALLER, "smaller", ""}
	tSum       = Tag{SUM, "+", ""}
	tSub       = Tag{SUB, "-", ""}
	tComment   = Tag{COMMENT, "--", "^\\-\\-"}
	tOBrk      = Tag{O_BRK, "{", "^{"}
	tCBrk      = Tag{C_BRK, "}", "^}"}
	tComma     = Tag{COMMA, ",", "^,"}
	tIf        = Tag{IF, "if", "^if"}
	tColon     = Tag{COLON, ":", "^:"}
	tSemicolon = Tag{SEMICOLON, ";", "^;"}
	tId        = Tag{ID, "", "^[A-Za-z][A-Za-z0-9-_?!><]*"}
)

var processableTags map[string]Tag

func init() {
	processableTags = map[string]Tag{
		IDENT:     tIdent,
		ASSIGN:    tAssign,
		O_PAREN:   tOParen,
		C_PAREN:   tCParen,
		EQUALS:    tEquals,
		DIFFERENT: tDifferent,
		BIGGER:    tBigger,
		SMALLER:   tSmaller,
		SUM:       tSum,
		SUB:       tSub,
		O_BRK:     tOBrk,
		C_BRK:     tCBrk,
		COMMA:     tComma,
		COLON:     tColon,
		SEMICOLON: tSemicolon,
		IF:        tIf,
		COMMENT:   tComment,
		ID:        tId,
	}
}

func GetProcessbleTags() map[string]Tag {
	return processableTags
}

func GetTag(c string) (Tag, error) {
	if t, has := processableTags[c]; has {
		return t, nil
	}
	return Tag{}, errors.New("Tag not found")
}
