package parser

import "blbx_lang/syntax/lexer"

// Lower interpolation to the existing string concat operation, which displays
// each value and evaluates arguments once in source order.
func (p *Parser) parseFormattedString() Node {
	start := p.Consume()
	receiver := NewToken("", STRING_LITERAL, start.Line, nil)
	member := NewToken("concat", NAMESPACE, start.Line, []Node{
		receiver, NewToken("concat", IDENTIFIER, start.Line, nil),
	})
	parts := []Node{member}
	for p.Current().Type != lexer.FSTRING_END && !p.AtEnd() {
		if p.Current().Type == lexer.STRING {
			parts = append(parts, p.parseString())
			continue
		}
		p.Expect(lexer.INTERPOLATION_START)
		parts = append(parts, p.parseExpression())
		p.Expect(lexer.INTERPOLATION_END)
	}
	p.Expect(lexer.FSTRING_END)
	return NewToken("concat", FUNCTION_CALL, start.Line, parts)
}
