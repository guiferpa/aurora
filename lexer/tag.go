package lexer

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
)

const (
	IDENT        = "IDENT"     // ident
	TAPE         = "TAPE"      // tape
	PULL         = "PULL"      // pull
	HEAD         = "HEAD"      // head
	TAIL         = "TAIL"      // tail
	PUSH         = "PUSH"      // push
	GLUE         = "GLUE"      // glue
	ASSIGN       = "ASSIGN"    // =
	O_PAREN      = "O_PAREN"   // (
	C_PAREN      = "C_PAREN"   // )
	EQUALS       = "EQUALS"    // equals
	DIFFERENT    = "DIFFERENT" // differenTag
	BIGGER       = "BIGGER"    // bigger
	SMALLER      = "SMALLER"   // smaller
	OR           = "OR"        // or
	AND          = "AND"       // and
	SUM          = "SUM"       // +
	SUB          = "SUB"       // -
	MULT         = "MULT"      // *
	DIV          = "DIV"       // /
	EXPO         = "EXPO"      // ^
	COMMENT_LINE = "COMMENT"   // #-
	O_BRK        = "O_BRK"     // [
	C_BRK        = "C_BRK"     // ]
	O_CUR_BRK    = "O_CUR_BRK" // {
	C_CUR_BRK    = "C_CUR_BRK" // }
	COMMA        = "COMMA"     // ,
	IF           = "IF"        // if
	ELSE         = "ELSE"      // else
	BRANCH       = "BRANCH"    // branch
	COLON        = "COLON"     // :
	SEMICOLON    = "SEMICOLON" // ;
	ID           = "ID"
	NUMBER       = "NUMBER"
	TRUE         = "TRUE"  // true
	FALSE        = "FALSE" // false
	WHITESPACE   = "WHITESPACE"
	BREAK_LINE   = "BREAK_LINE"
	PRINT        = "PRINT"     // print
	ARGUMENTS    = "ARGUMENTS" // arguments - It's responsible for get value from higher scopes
	EOF          = "EOF"
)

type Tag struct {
	Id          string
	Keyword     string
	Rule        string
	Description string
}

var (
	TagBreakLine  = Tag{BREAK_LINE, "", "^[\\r\\n]", ""}
	TagWhitespace = Tag{WHITESPACE, " ", "^[ ]+", ""}
	TagCallPrint  = Tag{PRINT, "print", "^print", "Print anything"}
	TagArguments  = Tag{ARGUMENTS, "arguments", "^arguments", "Get arguments from any callable scope"}
	TagIdent      = Tag{IDENT, "ident", "^ident", "Create an immutable identifier"}
	TagAssign     = Tag{ASSIGN, "=", "^=", ""}
	TagOParen     = Tag{O_PAREN, "(", "^\\(", ""}
	TagCParen     = Tag{C_PAREN, ")", "^\\)", ""}
	TagEquals     = Tag{EQUALS, "equals", "^equals", ""}
	TagDifferent  = Tag{DIFFERENT, "different", "^different", ""}
	TagBigger     = Tag{BIGGER, "bigger", "^bigger", ""}
	TagSmaller    = Tag{SMALLER, "smaller", "^smaller", ""}
	TagOr         = Tag{OR, "or", "^or", ""}
	TagAnd        = Tag{AND, "and", "^and", ""}
	TagSum        = Tag{SUM, "+", "^\\+", ""}
	TagSub        = Tag{SUB, "-", "^\\-", ""}
	TagMult       = Tag{MULT, "*", "^\\*", ""}
	TagDiv        = Tag{DIV, "/", "^\\/", ""}
	TagExpo       = Tag{EXPO, "^", "^\\^", ""}
	TagComment    = Tag{COMMENT_LINE, "#-", "^#-", ""}
	TagOBrk       = Tag{O_BRK, "[", "^\\[", ""}
	TagCBrk       = Tag{C_BRK, "]", "^\\]", ""}
	TagOCurBrk    = Tag{O_CUR_BRK, "{", "^{", ""}
	TagCCurBrk    = Tag{C_CUR_BRK, "}", "^}", ""}
	TagComma      = Tag{COMMA, ",", "^,", ""}
	TagIf         = Tag{IF, "if", "^if", "Make conditions with If"}
	TagElse       = Tag{ELSE, "else", "^else", "Make else for conditions with If"}
	TagColon      = Tag{COLON, ":", "^:", ""}
	TagBranch     = Tag{BRANCH, "branch", "^branch", "Make possible many branches"}
	TagSemicolon  = Tag{SEMICOLON, ";", "^;", ""}
	TagTrue       = Tag{TRUE, "true", "^true", ""}
	TagFalse      = Tag{FALSE, "false", "^false", ""}
	TagId         = Tag{ID, "", "^[A-Za-z][A-Za-z0-9-_?!><]*", ""}
	TagTape       = Tag{TAPE, "tape", "^tape", ""}
	TagHead       = Tag{HEAD, "head", "^head", "Get left to right nth items from a tape"}
	TagTail       = Tag{TAIL, "tail", "^tail", "Get right to left nth items from a tape"}
	TagPush       = Tag{PUSH, "push", "^push", "Push item in left to right"}
	TagPull       = Tag{PULL, "pull", "^pull", "Pull item in right to left"}
	TagGlue       = Tag{GLUE, "glue", "^glue", "Glue two values into just one"}
	TagNumber     = Tag{NUMBER, "", "^[0-9][0-9_]*\\b", ""}
	TagEOF        = Tag{EOF, "<EOF>", "", ""}
)

var processableTags = []Tag{
	TagWhitespace,
	TagBreakLine,
	TagComment,
	TagGlue,
	TagIf,
	TagElse,
	TagBranch,
	TagTape,
	TagPull,
	TagHead,
	TagTail,
	TagPush,
	TagIdent,
	TagAssign,
	TagOParen,
	TagCParen,
	TagEquals,
	TagDifferent,
	TagBigger,
	TagSmaller,
	TagOr,
	TagAnd,
	TagOBrk,
	TagCBrk,
	TagOCurBrk,
	TagCCurBrk,
	TagComma,
	TagColon,
	TagSemicolon,
	TagCallPrint,
	TagArguments,
	TagTrue,
	TagFalse,
	TagId,
	TagSum,
	TagSub,
	TagExpo,
	TagMult,
	TagDiv,
	TagNumber,
}

func GetProcessbleTags() []Tag {
	return processableTags
}

func GetProcessbleTagsWithDescription() []Tag {
	tags := make([]Tag, 0)
	for _, t := range GetProcessbleTags() {
		if strings.Compare(t.Description, "") == 0 {
			continue
		}
		tags = append(tags, t)
	}
	return tags
}

func MatchTagRule(bs []byte) (bool, Tag, []byte) {
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
