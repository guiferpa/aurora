package token

const (
	AND          = "AND"    // and
	AS           = "AS"     // as - names the shape a value is read with
	FEED         = "FEED"   // feed - reads a value applied to the scope
	ASSERT       = "ASSERT" // assert
	ASSIGN       = "ASSIGN" // =
	BIGGER       = "BIGGER" // bigger
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
	DOT          = "DOT"       // . - reads a field of a shape
	ELSE         = "ELSE"      // else
	EOF          = "EOF"
	EQUALS       = "EQUALS" // equals
	EXPO         = "EXPO"   // ^
	FALSE        = "FALSE"  // false
	HEAD         = "HEAD"   // head
	ID           = "ID"
	IDENT        = "IDENT" // ident
	IF           = "IF"    // if
	MULT         = "MULT"  // *
	NUMBER       = "NUMBER"
	O_BRK        = "O_BRK"     // [
	O_CUR_BRK    = "O_CUR_BRK" // {
	O_PAREN      = "O_PAREN"   // (
	OR           = "OR"        // or
	PRINTB       = "PRINTB"    // printb - the bytes of a value
	PRINTC       = "PRINTC"    // printc - the characters a value names
	PRINTD       = "PRINTD"    // printd - a value as a decimal number
	PULL         = "PULL"      // pull
	PUSH         = "PUSH"      // push
	RETURNS      = "RETURNS"   // returns - the shape a block answers with
	SEMICOLON    = "SEMICOLON" // ;
	SHAPE        = "SHAPE"     // shape - names the fields of a run of tapes
	SMALLER      = "SMALLER"   // smaller
	STRING       = "STRING"    // text literal "text" - one more way of writing a tape
	SUB          = "SUB"       // -
	SUM          = "SUM"       // +
	TAIL         = "TAIL"      // tail
	SLOAD        = "SLOAD"     // sload
	SSTORE       = "SSTORE"    // sstore
	CALLVALUE    = "CALLVALUE" // callvalue
	TRUE         = "TRUE"      // true
	USE          = "USE"       // use - brings a module in under an alias
	WHITESPACE   = "WHITESPACE"
)

type Tag struct {
	Id          string
	Keyword     string
	Description string
}

var (
	TagAnd        = Tag{AND, "and", ""}
	TagAs         = Tag{AS, "as", "Read a value with the shape of a shape"}
	TagFeed       = Tag{FEED, "feed", "Read the nth value fed to this scope"}
	TagAssert     = Tag{ASSERT, "assert", "Assert a condition in tests"}
	TagAssign     = Tag{ASSIGN, "=", ""}
	TagBigger     = Tag{BIGGER, "bigger", ""}
	TagBranch     = Tag{BRANCH, "branch", "Make possible many branches"}
	TagBreakLine  = Tag{BREAK_LINE, "", ""}
	TagCBrk       = Tag{C_BRK, "]", ""}
	TagCCurBrk    = Tag{C_CUR_BRK, "}", ""}
	TagCParen     = Tag{C_PAREN, ")", ""}
	TagPrintBytes = Tag{PRINTB, "printb", "Print the bytes of a value"}
	TagPrintChars = Tag{PRINTC, "printc", "Print a value as text"}
	TagPrintDec   = Tag{PRINTD, "printd", "Print a value as a decimal number"}
	TagColon      = Tag{COLON, ":", ""}
	TagComma      = Tag{COMMA, ",", ""}
	TagComment    = Tag{COMMENT_LINE, "#-", ""}
	TagDefer      = Tag{DEFER, "defer", "Defer scope execution (pointer to scope)"}
	TagDifferent  = Tag{DIFFERENT, "different", ""}
	TagDiv        = Tag{DIV, "/", ""}
	TagDot        = Tag{DOT, ".", ""}
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
	TagNumber     = Tag{NUMBER, "", ""}
	TagOBrk       = Tag{O_BRK, "[", ""}
	TagOCurBrk    = Tag{O_CUR_BRK, "{", ""}
	TagOParen     = Tag{O_PAREN, "(", ""}
	TagOr         = Tag{OR, "or", ""}
	TagPull       = Tag{PULL, "pull", "Pull item in right to left"}
	TagPush       = Tag{PUSH, "push", "Push item in left to right"}
	TagReturns    = Tag{RETURNS, "returns", "Name the shape a block returns"}
	TagSemicolon  = Tag{SEMICOLON, ";", ""}
	TagShape      = Tag{SHAPE, "shape", "Name the fields of a run of tapes"}
	TagSmaller    = Tag{SMALLER, "smaller", ""}
	TagString     = Tag{STRING, "", ""} // Text literal: "text", the bytes it holds, in a tape
	TagSub        = Tag{SUB, "-", ""}
	TagSum        = Tag{SUM, "+", ""}
	TagTail       = Tag{TAIL, "tail", "Get right to left nth items from a tape"}
	TagSload      = Tag{SLOAD, "sload", "Read what the chain keeps under a key"}
	TagSstore     = Tag{SSTORE, "sstore", "Keep a value under a key, between transactions"}
	TagCallValue  = Tag{CALLVALUE, "callvalue", "How much the transaction carried"}
	TagTrue       = Tag{TRUE, "true", ""}
	TagUse        = Tag{USE, "use", "Bring a module in under an alias"}
	TagWhitespace = Tag{WHITESPACE, " ", ""}
)

var processableTags = []Tag{
	TagPrintBytes,
	TagPrintChars,
	TagPrintDec,
	TagFeed,
	TagAssert,
	TagIdent,
	TagIf,
	TagElse,
	TagBranch,
	TagDefer,
	TagHead,
	TagTail,
	TagPush,
	TagPull,
	TagSload,
	TagSstore,
	TagCallValue,
	TagShape,
	TagAs,
	TagUse,
	TagReturns,
}

func GetProcessableTags() []Tag {
	return processableTags
}
