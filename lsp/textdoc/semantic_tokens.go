package textdoc

import (
	"encoding/json"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/lsp"
)

// Semantic tokens are how the language server colors Aurora in any client: instead of a
// per-editor grammar kept in sync by hand, the server reports what the lexer already
// knows. https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens

// Token types, in the order the client receives them in the legend. The index in this
// slice is what goes on the wire.
const (
	SemanticKeyword = iota
	SemanticNumber
	SemanticString
	SemanticComment
	SemanticOperator
	SemanticVariable
	SemanticFunction
)

// SemanticModifierDeclaration is bit 0 of the modifier set (the only modifier reported).
const SemanticModifierDeclaration = 1 << 0

// SemanticTokenTypes is the legend sent in the server capabilities.
var SemanticTokenTypes = []string{
	"keyword",
	"number",
	"string",
	"comment",
	"operator",
	"variable",
	"function",
}

// SemanticTokenModifiers is the modifier legend sent in the server capabilities.
var SemanticTokenModifiers = []string{"declaration"}

type SemanticTokensParams struct {
	TextDocument Identifier `json:"textDocument"`
}

type SemanticTokensRequest struct {
	lsp.Request
	Params SemanticTokensParams `json:"params"`
}

type SemanticTokensResult struct {
	Data []uint `json:"data"`
}

type SemanticTokensResponse struct {
	lsp.Response
	Result SemanticTokensResult `json:"result"`
}

func ParseSemanticTokensRequest(contents []byte) (*SemanticTokensRequest, error) {
	var req SemanticTokensRequest
	if err := json.Unmarshal(contents, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func NewSemanticTokensResponse(id int, data []uint) SemanticTokensResponse {
	return SemanticTokensResponse{
		Response: lsp.Response{RPC: "2.0", ID: &id},
		Result:   SemanticTokensResult{Data: data},
	}
}

// semanticToken is one highlighted span before delta encoding.
type semanticToken struct {
	offset    int // byte offset in the source
	length    int // byte length
	tokenType int
	modifiers int
}

// SemanticTokensFor lexes source and returns the delta-encoded token data the LSP expects.
//
// It uses the raw token stream (GetTokens) rather than GetFilledTokens because whitespace
// and line breaks are needed to place comments, and a lexer error still yields the tokens
// read so far — so a file being typed keeps its colors instead of going blank.
func SemanticTokensFor(source string) []uint {
	tokens, _ := lexer.New(lexer.NewLexerOptions{}).GetTokens([]byte(source))
	mapper := lsp.NewMapper(source)
	return encodeSemanticTokens(mapper, collectSemanticTokens(mapper, tokens))
}

func collectSemanticTokens(mapper *lsp.Mapper, tokens []lexer.Token) []semanticToken {
	out := make([]semanticToken, 0, len(tokens))
	for i, token := range tokens {
		tag := token.GetTag().Id
		length := len(token.GetMatch())
		offset := token.GetCursor()

		if tag == lexer.COMMENT_LINE {
			// The lexer drops everything after "#-", so the comment body comes from the text.
			end := mapper.LineEndOffset(offset)
			out = append(out, semanticToken{offset: offset, length: end - offset, tokenType: SemanticComment})
			continue
		}

		tokenType, ok := semanticTypeOf(tag)
		if !ok {
			continue
		}

		modifiers := 0
		if tag == lexer.ID {
			tokenType, modifiers = classifyIdentifier(tokens, i)
		}
		if length == 0 {
			continue
		}
		out = append(out, semanticToken{offset: offset, length: length, tokenType: tokenType, modifiers: modifiers})
	}
	return out
}

// semanticTypeOf maps a lexer tag to a token type. Tags with no visual meaning
// (whitespace, line breaks, EOF, punctuation) report false and are skipped.
func semanticTypeOf(tag string) (int, bool) {
	switch tag {
	case lexer.IDENT, lexer.IF, lexer.ELSE, lexer.BRANCH, lexer.DEFER,
		lexer.PRINTB, lexer.PRINTC, lexer.PRINTD, lexer.ASSERT, lexer.FEED,
		lexer.HEAD, lexer.TAIL, lexer.PUSH, lexer.PULL, lexer.TRUE, lexer.FALSE:
		return SemanticKeyword, true
	case lexer.NUMBER:
		return SemanticNumber, true
	case lexer.STRING:
		return SemanticString, true
	case lexer.SUM, lexer.SUB, lexer.MULT, lexer.DIV, lexer.EXPO, lexer.ASSIGN,
		lexer.EQUALS, lexer.DIFFERENT, lexer.BIGGER, lexer.SMALLER, lexer.AND, lexer.OR:
		return SemanticOperator, true
	case lexer.ID:
		return SemanticVariable, true
	default:
		return 0, false
	}
}

// classifyIdentifier refines an ID by looking at its neighbours: "name(" is a call and
// "ident name" is a declaration.
func classifyIdentifier(tokens []lexer.Token, i int) (tokenType, modifiers int) {
	if next := nextMeaningful(tokens, i); next != nil && next.GetTag().Id == lexer.O_PAREN {
		return SemanticFunction, 0
	}
	if prev := prevMeaningful(tokens, i); prev != nil && prev.GetTag().Id == lexer.IDENT {
		return SemanticVariable, SemanticModifierDeclaration
	}
	return SemanticVariable, 0
}

func nextMeaningful(tokens []lexer.Token, i int) lexer.Token {
	for j := i + 1; j < len(tokens); j++ {
		if isLayout(tokens[j].GetTag().Id) {
			continue
		}
		return tokens[j]
	}
	return nil
}

func prevMeaningful(tokens []lexer.Token, i int) lexer.Token {
	for j := i - 1; j >= 0; j-- {
		if isLayout(tokens[j].GetTag().Id) {
			continue
		}
		return tokens[j]
	}
	return nil
}

func isLayout(tag string) bool {
	return tag == lexer.WHITESPACE || tag == lexer.BREAK_LINE
}

// encodeSemanticTokens turns absolute spans into the protocol's delta encoding:
// five integers per token (deltaLine, deltaStartChar, length, type, modifiers), each
// position relative to the previous token. Spans crossing a line break are split so no
// entry ever spans lines, which the encoding cannot express.
func encodeSemanticTokens(mapper *lsp.Mapper, tokens []semanticToken) []uint {
	data := make([]uint, 0, len(tokens)*5)
	lastLine, lastChar := 0, 0

	for _, token := range tokens {
		offset, remaining := token.offset, token.length
		for remaining > 0 {
			lineEnd := mapper.LineEndOffset(offset)
			length := remaining
			if offset+length > lineEnd {
				length = lineEnd - offset
			}
			if length <= 0 {
				break
			}

			start := mapper.Position(offset)
			end := mapper.Position(offset + length)

			deltaLine := start.Line - lastLine
			deltaChar := start.Character
			if deltaLine == 0 {
				deltaChar = start.Character - lastChar
			}

			data = append(data,
				uint(deltaLine),
				uint(deltaChar),
				uint(end.Character-start.Character),
				uint(token.tokenType),
				uint(token.modifiers),
			)

			lastLine, lastChar = start.Line, start.Character
			remaining -= length
			offset += length
			// Skip the line break itself before continuing on the next line.
			for offset < len(mapper.Text()) && (mapper.Text()[offset] == '\r' || mapper.Text()[offset] == '\n') {
				offset++
				remaining--
			}
		}
	}

	return data
}
