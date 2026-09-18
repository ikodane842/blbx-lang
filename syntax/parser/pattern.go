package parser

import (
	"blbx_lang/syntax/diagnostic"
	"blbx_lang/syntax/lexer"
)

func (p *Parser) startsDestructuring() bool {
	if p.Current().Type != lexer.OPEN_BRACKET && p.Current().Type != lexer.OPEN_BRACE {
		return false
	}
	depth := 0
	for index := p.Pos; index < len(p.Input); index++ {
		switch p.Input[index].Type {
		case lexer.OPEN_BRACKET, lexer.OPEN_BRACE, lexer.OPEN_PAREN:
			depth++
		case lexer.CLOSED_BRACKET, lexer.CLOSED_BRACE, lexer.CLOSED_PAREN:
			depth--
			if depth == 0 {
				return index+1 < len(p.Input) && p.Input[index+1].Type == lexer.ASSIGN && p.Input[index+1].Name == "="
			}
		}
	}
	return false
}

func (p *Parser) parsePattern(names map[string]bool) Node {
	defer p.enter()()
	tok := p.Current()
	if tok.Type == lexer.IDENTIFIER {
		if tok.Name == "null" || tok.Name == "true" || tok.Name == "false" {
			p.fail(diagnostic.InvalidTarget, "pattern target must be a variable name")
		}
		if names[tok.Name] {
			p.fail(diagnostic.DuplicateBinding, "duplicate binding in destructuring pattern")
		}
		names[tok.Name] = true
		return p.parseIdentifier()
	}
	kind, end := ARRAY_PATTERN, lexer.CLOSED_BRACKET
	if tok.Type == lexer.OPEN_BRACE {
		kind, end = OBJECT_PATTERN, lexer.CLOSED_BRACE
	} else if tok.Type != lexer.OPEN_BRACKET {
		p.fail(diagnostic.InvalidTarget, "expected binding name, array pattern, or object pattern")
	}
	p.Consume()
	children := []Node{}
	keys := map[string]bool{}
	for !p.AtEnd() && p.Current().Type != end {
		if p.Current().Type == lexer.DOT && p.Peek(1).Type == lexer.DOT && p.Peek(2).Type == lexer.DOT {
			rest := p.Consume()
			p.Consume()
			p.Consume()
			if p.Current().Type != lexer.IDENTIFIER {
				p.fail(diagnostic.InvalidTarget, "rest target must be a variable name")
			}
			target := p.parsePattern(names)
			children = append(children, NewToken("rest", REST_PATTERN, rest.Line, []Node{target}))
			p.match(lexer.COMMA)
			if p.Current().Type != end {
				p.fail(diagnostic.RestPosition, "rest binding must be last")
			}
			break
		}
		if kind == ARRAY_PATTERN {
			children = append(children, p.parsePattern(names))
		} else {
			key := p.Current()
			if key.Type != lexer.IDENTIFIER && key.Type != lexer.STRING {
				p.fail(diagnostic.InvalidTarget, "expected object field name")
			}
			if keys[key.Name] {
				p.fail(diagnostic.DuplicateKey, "duplicate key in object pattern")
			}
			keys[key.Name] = true
			var target Node
			if p.Peek(1).Type == lexer.ASSIGN && p.Peek(1).Name == ":" {
				p.Consume()
				p.Consume()
				target = p.parsePattern(names)
			} else {
				target = p.parsePattern(names)
			}
			children = append(children, NewToken(key.Name, PATTERN_FIELD, key.Line, []Node{target}))
		}
		if !p.match(lexer.COMMA) {
			break
		}
	}
	p.Expect(end)
	return NewToken("pattern", kind, tok.Line, children)
}
