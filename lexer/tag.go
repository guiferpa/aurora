package lexer

const (
	AND          = "AND"       // and
	ARGUMENTS    = "ARGUMENTS" // arguments - It's responsible for get value from higher scopes
	AS           = "AS"       // as
	ASSERT       = "ASSERT"    // assert
	ASSIGN       = "ASSIGN"    // =
	BIGGER       = "BIGGER"    // bigger
	BREAK_LINE   = "BREAK_LINE"
	BRANCH       = "BRANCH"    // branch
	C_BRK        = "C_BRK"     // ]
	C_CUR_BRK    = "C_CUR_BRK" // }
	C_PAREN      = "C_PAREN"   // )
	COLON        = "COLON"     // :
	COMMA        = "COMMA"     // ,
	COMMENT_LINE = "COMMENT"   // #-
	DEFER        = "DEFER"     // defer - delayed scope execution
	DIFFERENT    = "DIFFERENT" // differenTag
	DIV          = "DIV"       // /
	ECHO         = "ECHO"      // echo - print bytes as text
	ELSE         = "ELSE"      // else
	EOF          = "EOF"
	EQUALS       = "EQUALS" // equals
	EXPO         = "EXPO"   // ^
	FALSE        = "FALSE"  // false
	HEAD         = "HEAD"   // head
	ID           = "ID"
	IDENT        = "IDENT"    // ident
	IF           = "IF"       // if
	MULT         = "MULT"     // *
	NOTHING      = "NOTHING"  // nothing
	NS_SCOPE     = "NS_SCOPE" // :: Namespace scope operator
	NUMBER       = "NUMBER"
	O_BRK        = "O_BRK"     // [
	O_CUR_BRK    = "O_CUR_BRK" // {
	O_PAREN      = "O_PAREN"   // (
	OR           = "OR"        // or
	PRINT        = "PRINT"     // print
	PULL         = "PULL"      // pull
	PUSH         = "PUSH"      // push
	SEMICOLON    = "SEMICOLON" // ;
	SMALLER      = "SMALLER"   // smaller
	STRING       = "STRING"    // string literal "text" (reel - array of tapes)
	SUB          = "SUB"       // -
	SUM          = "SUM"       // +
	TAIL         = "TAIL"      // tail
	TRUE         = "TRUE"      // true
	USE          = "USE"       // use
	WHITESPACE   = "WHITESPACE"
)

type Tag struct {
	Id          string
	Keyword     string
	Description string
}

var (
	TagAnd        = Tag{AND, "and", ""}
	TagArguments  = Tag{ARGUMENTS, "arguments", "Get arguments from any callable scope"}
	TagAs         = Tag{AS, "as", "Alias for use: use path as name"}
	TagAssert     = Tag{ASSERT, "assert", "Assert a condition in tests"}
	TagAssign     = Tag{ASSIGN, "=", ""}
	TagBigger     = Tag{BIGGER, "bigger", ""}
	TagBranch     = Tag{BRANCH, "branch", "Make possible many branches"}
	TagBreakLine  = Tag{BREAK_LINE, "", ""}
	TagCBrk       = Tag{C_BRK, "]", ""}
	TagCCurBrk    = Tag{C_CUR_BRK, "}", ""}
	TagCParen     = Tag{C_PAREN, ")", ""}
	TagCallPrint  = Tag{PRINT, "print", "Print anything"}
	TagColon      = Tag{COLON, ":", ""}
	TagComma      = Tag{COMMA, ",", ""}
	TagComment    = Tag{COMMENT_LINE, "#-", ""}
	TagDefer      = Tag{DEFER, "defer", "Defer scope execution (pointer to scope)"}
	TagDifferent  = Tag{DIFFERENT, "different", ""}
	TagDiv        = Tag{DIV, "/", ""}
	TagEcho       = Tag{ECHO, "echo", "Echo bytes as text"}
	TagElse       = Tag{ELSE, "else", "Make else for conditions with If"}
	TagEOF        = Tag{EOF, "<EOF>", ""}
	TagEquals     = Tag{EQUALS, "equals", ""}
	TagExpo       = Tag{EXPO, "^", ""}
	TagFalse      = Tag{FALSE, "false", ""}
	TagHead       = Tag{HEAD, "head", "Get left to right nth items from a tape"}
	TagId         = Tag{ID, "", ""}
	TagIdent      = Tag{IDENT, "ident", "Create an immutable identifier"}
	TagIf         = Tag{IF, "if", "Make conditions with If"}
	TagMult       = Tag{MULT, "*", ""}
	TagNothing    = Tag{NOTHING, "nothing", "Universal neutral value (8 zero bytes)"}
	TagNumber     = Tag{NUMBER, "", ""}
	TagOBrk       = Tag{O_BRK, "[", ""}
	TagOCurBrk    = Tag{O_CUR_BRK, "{", ""}
	TagOParen     = Tag{O_PAREN, "(", ""}
	TagOr         = Tag{OR, "or", ""}
	TagPull       = Tag{PULL, "pull", "Pull item in right to left"}
	TagPush       = Tag{PUSH, "push", "Push item in left to right"}
	TagNsScope    = Tag{NS_SCOPE, "::", "Namespace scope operator"}
	TagSemicolon  = Tag{SEMICOLON, ";", ""}
	TagSmaller    = Tag{SMALLER, "smaller", ""}
	TagString     = Tag{STRING, "", ""} // String literal: "text" (reel - array of tapes)
	TagSub        = Tag{SUB, "-", ""}
	TagSum        = Tag{SUM, "+", ""}
	TagTail       = Tag{TAIL, "tail", "Get right to left nth items from a tape"}
	TagTrue       = Tag{TRUE, "true", ""}
	TagUse        = Tag{USE, "use", "Import path into scope: use path as name (e.g. use utils::fs::io as io)"}
	TagWhitespace = Tag{WHITESPACE, " ", ""}
)

var processableTags = []Tag{
	TagCallPrint,
	TagEcho,
	TagArguments,
	TagAssert,
	TagAs,
	TagIdent,
	TagIf,
	TagElse,
	TagBranch,
	TagDefer,
	TagNothing,
	TagHead,
	TagTail,
	TagPush,
	TagPull,
	TagUse,
}

func GetProcessableTags() []Tag {
	return processableTags
}

func MatchToken(bs []byte) (bool, Tag, []byte) {
	return ScanToken(bs)
}
