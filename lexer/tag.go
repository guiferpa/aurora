package lexer

const (
	IDENT        = "IDENT"     // ident
	PULL         = "PULL"      // pull
	HEAD         = "HEAD"      // head
	TAIL         = "TAIL"      // tail
	PUSH         = "PUSH"      // push
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
	STRING       = "STRING" // string literal "text" (reel - array of tapes)
	TRUE         = "TRUE"   // true
	FALSE        = "FALSE"  // false
	WHITESPACE   = "WHITESPACE"
	BREAK_LINE   = "BREAK_LINE"
	PRINT        = "PRINT"     // print
	ECHO         = "ECHO"      // echo - print bytes as text
	ARGUMENTS    = "ARGUMENTS" // arguments - It's responsible for get value from higher scopes
	ASSERT       = "ASSERT"    // assert
	EOF          = "EOF"
)

type Tag struct {
	Id          string
	Keyword     string
	Description string
}

var (
	TagBreakLine  = Tag{BREAK_LINE, "", ""}
	TagWhitespace = Tag{WHITESPACE, " ", ""}
	TagCallPrint  = Tag{PRINT, "print", "Print anything"}
	TagEcho       = Tag{ECHO, "echo", "Echo bytes as text"}
	TagArguments  = Tag{ARGUMENTS, "arguments", "Get arguments from any callable scope"}
	TagAssert     = Tag{ASSERT, "assert", "Assert a condition in tests"}
	TagIdent      = Tag{IDENT, "ident", "Create an immutable identifier"}
	TagAssign     = Tag{ASSIGN, "=", ""}
	TagOParen     = Tag{O_PAREN, "(", ""}
	TagCParen     = Tag{C_PAREN, ")", ""}
	TagEquals     = Tag{EQUALS, "equals", ""}
	TagDifferent  = Tag{DIFFERENT, "different", ""}
	TagBigger     = Tag{BIGGER, "bigger", ""}
	TagSmaller    = Tag{SMALLER, "smaller", ""}
	TagOr         = Tag{OR, "or", ""}
	TagAnd        = Tag{AND, "and", ""}
	TagSum        = Tag{SUM, "+", ""}
	TagSub        = Tag{SUB, "-", ""}
	TagMult       = Tag{MULT, "*", ""}
	TagDiv        = Tag{DIV, "/", ""}
	TagExpo       = Tag{EXPO, "^", ""}
	TagComment    = Tag{COMMENT_LINE, "#-", ""}
	TagOBrk       = Tag{O_BRK, "[", ""}
	TagCBrk       = Tag{C_BRK, "]", ""}
	TagOCurBrk    = Tag{O_CUR_BRK, "{", ""}
	TagCCurBrk    = Tag{C_CUR_BRK, "}", ""}
	TagComma      = Tag{COMMA, ",", ""}
	TagIf         = Tag{IF, "if", "Make conditions with If"}
	TagElse       = Tag{ELSE, "else", "Make else for conditions with If"}
	TagColon      = Tag{COLON, ":", ""}
	TagBranch     = Tag{BRANCH, "branch", "Make possible many branches"}
	TagSemicolon  = Tag{SEMICOLON, ";", ""}
	TagTrue       = Tag{TRUE, "true", ""}
	TagFalse      = Tag{FALSE, "false", ""}
	TagId         = Tag{ID, "", ""}
	TagHead       = Tag{HEAD, "head", "Get left to right nth items from a tape"}
	TagTail       = Tag{TAIL, "tail", "Get right to left nth items from a tape"}
	TagPush       = Tag{PUSH, "push", "Push item in left to right"}
	TagPull       = Tag{PULL, "pull", "Pull item in right to left"}
	TagNumber     = Tag{NUMBER, "", ""}
	TagString     = Tag{STRING, "", ""} // String literal: "text" (reel - array of tapes)
	TagEOF        = Tag{EOF, "<EOF>", ""}
)

var processableTags = []Tag{
	TagCallPrint,
	TagEcho,
	TagArguments,
	TagAssert,
	TagIdent,
	TagIf,
	TagElse,
	TagBranch,
	TagHead,
	TagTail,
	TagPush,
	TagPull,
}

func GetProcessableTags() []Tag {
	return processableTags
}

func MatchToken(bs []byte) (bool, Tag, []byte) {
	return ScanToken(bs)
}
