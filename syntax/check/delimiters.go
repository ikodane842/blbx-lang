package check

import (
	"blbx_lang/syntax/diagnostic"
	"blbx_lang/syntax/lexer"
	"fmt"
)

func checkDelimiters(tokens []lexer.LexerToken) []diagnostic.Diagnostic {
	stack := []lexer.LexerToken{}
	out := []diagnostic.Diagnostic{}
	report := func(t lexer.LexerToken, message string) {
		out = append(out, diagnostic.Diagnostic{Line: t.Line, Column: t.Column, EndLine: t.EndLine, EndColumn: t.EndColumn, Severity: "error", Code: diagnostic.Delimiter, Message: message})
	}
	pairs := map[lexer.TokenType]lexer.TokenType{lexer.CLOSED_PAREN: lexer.OPEN_PAREN, lexer.CLOSED_BRACKET: lexer.OPEN_BRACKET, lexer.CLOSED_BRACE: lexer.OPEN_BRACE}
	for _, t := range tokens {
		switch t.Type {
		case lexer.OPEN_PAREN, lexer.OPEN_BRACKET, lexer.OPEN_BRACE:
			stack = append(stack, t)
		default:
			if opening, closing := pairs[t.Type]; closing {
				if len(stack) == 0 {
					report(t, fmt.Sprintf("unmatched closing delimiter %q", t.Name))
					continue
				}
				top := stack[len(stack)-1]
				if top.Type != opening {
					report(t, fmt.Sprintf("misplaced delimiter %q; does not close %q at %d:%d", t.Name, top.Name, top.Line, top.Column))
				}
				stack = stack[:len(stack)-1]
			}
		}
	}
	for _, t := range stack {
		report(t, fmt.Sprintf("unclosed delimiter %q", t.Name))
	}
	return out
}
