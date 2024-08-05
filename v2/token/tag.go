package token

import "errors"

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
	tComment   = Tag{COMMENT, "--", ""}
	tOBrk      = Tag{O_BRK, "{", ""}
	tCBrk      = Tag{C_BRK, "}", ""}
	tComma     = Tag{COMMA, ",", ""}
	tIf        = Tag{IF, "if", ""}
	tColon     = Tag{COLON, ":", ""}
	tSemicolon = Tag{SEMICOLON, ";", ""}
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
		COMMENT:   tComment,
		O_BRK:     tOBrk,
		C_BRK:     tCBrk,
		COMMA:     tComma,
		IF:        tIf,
		COLON:     tColon,
		SEMICOLON: tSemicolon,
	}
}

func GetTag(c string) (Tag, error) {
	if t, has := processableTags[c]; has {
		return t, nil
	}
	return Tag{}, errors.New("Tag not found")
}
