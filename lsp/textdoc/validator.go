package textdoc

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/lsp"
	"github.com/guiferpa/aurora/parser"
)

// Severity levels for diagnostics
const (
	SeverityError       = 1
	SeverityWarning     = 2
	SeverityInformation = 3
	SeverityHint        = 4
)

// ValidateCode validates Aurora code and returns diagnostics
func ValidateCode(source string) Diagnostics {
	diagnostics := Diagnostics{}

	// Tokenize the source code
	tokens, err := lexer.New(lexer.NewLexerOptions{
		EnableLogging: false,
	}).GetFilledTokens([]byte(source))
	if err != nil {
		// Parse lexer error to extract line and column
		line, col := parseErrorPosition(err.Error())
		diagnostics = append(diagnostics, Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      line - 1, // LSP uses 0-based line numbers
					Character: col - 1,  // LSP uses 0-based column numbers
				},
				End: lsp.Position{
					Line:      line - 1,
					Character: col,
				},
			},
			Severity: SeverityError,
			Source:   "aurora-lexer",
			Message:  err.Error(),
		})
		return diagnostics
	}

	// Parse the tokens
	p := parser.New(tokens, parser.NewParserOptions{
		Filename:      "",
		EnableLogging: false,
	})
	_, err = p.Parse()
	if err != nil {
		// Parse parser error to extract line and column
		line, col := parseErrorPosition(err.Error())
		diagnostics = append(diagnostics, Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      line - 1, // LSP uses 0-based line numbers
					Character: col - 1,  // LSP uses 0-based column numbers
				},
				End: lsp.Position{
					Line:      line - 1,
					Character: col,
				},
			},
			Severity: SeverityError,
			Source:   "aurora-parser",
			Message:  err.Error(),
		})
		return diagnostics
	}

	return diagnostics
}

// parseErrorPosition extracts line and column from error messages
// Handles formats like:
// - "unexpected character at line 5, column 10"
// - "unexpected token X at line 3 and column 5"
// - "missing identifier name at line: 2, column 10"
func parseErrorPosition(errMsg string) (line, col int) {
	// Try to find line number
	lineRegex := regexp.MustCompile(`line[:\s]+(\d+)`)
	lineMatch := lineRegex.FindStringSubmatch(errMsg)
	if len(lineMatch) > 1 {
		if l, err := strconv.Atoi(lineMatch[1]); err == nil {
			line = l
		}
	}

	// Try to find column number
	colRegex := regexp.MustCompile(`column[:\s]+(\d+)`)
	colMatch := colRegex.FindStringSubmatch(errMsg)
	if len(colMatch) > 1 {
		if c, err := strconv.Atoi(colMatch[1]); err == nil {
			col = c
		}
	}

	// If we couldn't parse, default to line 1, column 1
	if line == 0 {
		line = 1
	}
	if col == 0 {
		col = 1
	}

	return line, col
}

// GetTokenAtPosition finds the token at a given position in the source
func GetTokenAtPosition(source string, pos lsp.Position) (lexer.Token, error) {
	tokens, err := lexer.New(lexer.NewLexerOptions{
		EnableLogging: false,
	}).GetFilledTokens([]byte(source))
	if err != nil {
		return nil, err
	}

	// LSP positions are 0-based, our tokens are 1-based
	targetLine := pos.Line + 1
	targetCol := pos.Character + 1

	for _, token := range tokens {
		tokenLine := token.GetLine()
		tokenCol := token.GetColumn()
		tokenMatch := token.GetMatch()

		// Check if position is within this token
		if tokenLine == targetLine {
			if targetCol >= tokenCol && targetCol < tokenCol+len(tokenMatch) {
				return token, nil
			}
		}
	}

	return nil, nil
}

// GetLineContent returns the content of a specific line (0-based)
func GetLineContent(source string, line int) string {
	lines := strings.Split(source, "\n")
	if line >= 0 && line < len(lines) {
		return lines[line]
	}
	return ""
}

// GetHoverInfo returns hover information for a position in the source code
func GetHoverInfo(source string, pos lsp.Position) string {
	token, err := GetTokenAtPosition(source, pos)
	if err != nil || token == nil {
		return ""
	}

	tag := token.GetTag()
	match := string(token.GetMatch())

	// Handle keywords
	if tag.Description != "" {
		return tag.Description
	}

	// Handle identifiers - try to find their definition
	if tag.Id == lexer.ID {
		// Try to parse and find the identifier definition
		tokens, err := lexer.New(lexer.NewLexerOptions{
			EnableLogging: false,
		}).GetFilledTokens([]byte(source))
		if err != nil {
			return ""
		}
		p := parser.New(tokens, parser.NewParserOptions{
			Filename:      "",
			EnableLogging: false,
		})
		ast, err := p.Parse()
		if err != nil {
			return ""
		}

		// Find the identifier definition
		def := findIdentifierDefinition(ast, match)
		if def != nil {
			return formatIdentifierInfo(match, def)
		}

		return "identifier: " + match
	}

	// Handle numbers
	if tag.Id == lexer.NUMBER {
		return "number: " + match
	}

	// Handle boolean literals
	if tag.Id == lexer.TRUE {
		return "boolean: true"
	}
	if tag.Id == lexer.FALSE {
		return "boolean: false"
	}

	return ""
}

// findIdentifierDefinition finds the definition of an identifier in the AST
func findIdentifierDefinition(ast parser.AST, name string) *parser.IdentStatementNode {
	return findIdentifierInStatements(ast.Module.Statements, name)
}

func findIdentifierInStatements(stmts []parser.Node, name string) *parser.IdentStatementNode {
	for _, stmt := range stmts {
		if identStmt, ok := stmt.(parser.IdentStatementNode); ok {
			if identStmt.Id == name {
				return &identStmt
			}
		}
		if stmtNode, ok := stmt.(parser.StatementNode); ok {
			if identStmt, ok := stmtNode.Node.(parser.IdentStatementNode); ok {
				if identStmt.Id == name {
					return &identStmt
				}
			}
		}
		// Recursively search in nested structures
		if result := findIdentifierInNode(stmt, name); result != nil {
			return result
		}
	}
	return nil
}

func findIdentifierInNode(node parser.Node, name string) *parser.IdentStatementNode {
	switch n := node.(type) {
	case parser.BlockExpressionNode:
		return findIdentifierInStatements(n.Body, name)
	case parser.DeferExpressionNode:
		return findIdentifierInStatements(n.Block.Body, name)
	case parser.IfExpressionNode:
		if result := findIdentifierInStatements(n.Body, name); result != nil {
			return result
		}
		if n.Else != nil {
			return findIdentifierInStatements(n.Else.Body, name)
		}
	}
	return nil
}

// formatIdentifierInfo formats information about an identifier definition
func formatIdentifierInfo(name string, def *parser.IdentStatementNode) string {
	exprType := getExpressionType(def.Expression)
	return "identifier: " + name + "\n" + "type: " + exprType
}

// getExpressionType returns a string representation of the expression type
func getExpressionType(expr parser.Node) string {
	switch expr.(type) {
	case parser.NumberLiteralNode:
		return "number"
	case parser.BooleanLiteral:
		return "boolean"
	case parser.IdLiteralNode:
		return "identifier"
	case parser.TapeBracketExpression:
		return "tape (array)"
	case parser.PullExpression:
		return "tape (pull operation)"
	case parser.PushExpression:
		return "tape (push operation)"
	case parser.HeadExpression:
		return "tape (head operation)"
	case parser.TailExpression:
		return "tape (tail operation)"
	case parser.BinaryExpressionNode:
		return "binary expression"
	case parser.RelativeExpression:
		return "relative expression"
	case parser.BooleanExpression:
		return "boolean expression"
	case parser.IfExpressionNode:
		return "if expression"
	case parser.BlockExpressionNode:
		return "block expression"
	case parser.CalleeLiteralNode:
		return "function call"
	default:
		return "expression"
	}
}

// FindIdentifierDefinition finds the definition location of an identifier
func FindIdentifierDefinition(source string, name string) (lsp.Position, bool) {
	tokens, err := lexer.New(lexer.NewLexerOptions{
		EnableLogging: false,
	}).GetFilledTokens([]byte(source))
	if err != nil {
		return lsp.Position{}, false
	}

	p := parser.New(tokens, parser.NewParserOptions{
		Filename:      "",
		EnableLogging: false,
	})
	ast, err := p.Parse()
	if err != nil {
		return lsp.Position{}, false
	}

	def := findIdentifierDefinition(ast, name)
	if def == nil {
		return lsp.Position{}, false
	}

	// Get position from the token
	token := def.Token
	if token == nil {
		return lsp.Position{}, false
	}

	return lsp.Position{
		Line:      token.GetLine() - 1, // LSP uses 0-based
		Character: token.GetColumn() - 1,
	}, true
}
