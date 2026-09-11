package parser

import "blbx_lang/syntax/lexer"

type Parser struct {
	Input  []lexer.LexerToken
	Output []Node
	Pos    int
}

func (p *Parser) Consume() lexer.LexerToken {
	token := p.Input[p.Pos]
	p.Pos++
	return token
}

func (p *Parser) AtEnd() bool {
	return p.Pos >= len(p.Input)
}

func (p *Parser) Current() lexer.LexerToken {
	if p.AtEnd() {
		return lexer.LexerToken{}
	}

	return p.Input[p.Pos]
}

func (p *Parser) Peek(offset int) lexer.LexerToken {
	if p.Pos+offset >= len(p.Input) {
		return lexer.LexerToken{}
	}

	return p.Input[p.Pos+offset]
}

func (p *Parser) Get() []Node {
	return p.Output
}

func (p *Parser) Set(tokens []lexer.LexerToken) {
	p.Input = tokens
	p.Pos = 0
	p.Output = []Node{}
}

func (p *Parser) Expect(tt lexer.TokenType) {
	if p.AtEnd() || p.Current().Type != tt {
		return
	}

	p.Consume()
}

func (p *Parser) match(tt lexer.TokenType) bool {
	if p.AtEnd() || p.Current().Type != tt {
		return false
	}

	p.Consume()
	return true
}

func (p *Parser) parseString() Node {
	tok := p.Consume()

	return NewToken(
		tok.Name,
		STRING_LITERAL,
		tok.Line,
		[]Node{},
	)
}

func (p *Parser) parseInteger() Node {
	tok := p.Consume()

	return NewToken(
		tok.Name,
		INTEGER_LITERAL,
		tok.Line,
		[]Node{},
	)
}

func (p *Parser) parseFloat() Node {
	tok := p.Consume()

	return NewToken(
		tok.Name,
		FLOAT_LITERAL,
		tok.Line,
		[]Node{},
	)
}

func (p *Parser) parseBoolean() Node {
	tok := p.Consume()

	return NewToken(
		tok.Name,
		BOOLEAN_LITERAL,
		tok.Line,
		[]Node{},
	)
}

func (p *Parser) parseIdentifier() Node {
	tok := p.Consume()

	return NewToken(
		tok.Name,
		IDENTIFIER,
		tok.Line,
		[]Node{},
	)
}

func (p *Parser) parseArguments() []Node {
	args := []Node{}

	for !p.AtEnd() && p.Current().Type != lexer.CLOSED_PAREN {
		start := p.Pos
		arg := p.parseStatement()
		if arg.Type != ILLEGAL {
			args = append(args, arg)
		}

		if p.Pos == start {
			p.Consume()
		}

		if !p.match(lexer.COMMA) {
			break
		}
	}

	p.Expect(lexer.CLOSED_PAREN)
	return args
}

func (p *Parser) parseFunctionCall() Node {
	tok := p.Consume()
	p.Expect(lexer.OPEN_PAREN)

	return Node{
		Name:     tok.Name,
		Type:     FUNCTION_CALL,
		Line:     tok.Line,
		Children: p.parseArguments(),
	}
}

func (p *Parser) parseArrayLiteral() Node {
	tok := p.Consume()
	items := []Node{}

	for !p.AtEnd() && p.Current().Type != lexer.CLOSED_BRACKET {
		start := p.Pos
		item := p.parseExpression()
		if item.Type != ILLEGAL {
			items = append(items, item)
		}

		if p.Pos == start {
			p.Consume()
		}

		if !p.match(lexer.COMMA) {
			break
		}
	}

	p.Expect(lexer.CLOSED_BRACKET)
	return NewToken("array", ARRAY_LITERAL, tok.Line, items)
}

func (p *Parser) parseBlock() Node {
	tok := p.Consume()
	body := []Node{}

	for !p.AtEnd() && p.Current().Type != lexer.CLOSED_BRACE {
		start := p.Pos
		stmt := p.parseStatement()
		if stmt.Type != ILLEGAL {
			body = append(body, stmt)
		}

		if p.Pos == start {
			p.Consume()
		}
	}

	p.Expect(lexer.CLOSED_BRACE)
	return NewToken("block", BLOCK, tok.Line, body)
}

func (p *Parser) parseParenExpression() Node {
	tok := p.Consume()
	items := []Node{}

	for !p.AtEnd() && p.Current().Type != lexer.CLOSED_PAREN {
		start := p.Pos
		item := p.parseStatement()
		if item.Type != ILLEGAL {
			items = append(items, item)
		}

		if p.Pos == start {
			p.Consume()
		}

		if !p.match(lexer.COMMA) {
			break
		}
	}

	p.Expect(lexer.CLOSED_PAREN)

	if len(items) == 1 {
		return items[0]
	}

	return NewToken("tuple", TUPLE_LITERAL, tok.Line, items)
}

func (p *Parser) parsePrimary() Node {
	if p.AtEnd() {
		return Node{Type: ILLEGAL}
	}

	switch p.Current().Type {
	case lexer.IDENTIFIER:
		if p.Current().Name == "true" || p.Current().Name == "false" {
			return p.parseBoolean()
		}

		if p.Peek(1).Type == lexer.OPEN_PAREN {
			return p.parseFunctionCall()
		}

		return p.parseIdentifier()
	case lexer.STRING:
		return p.parseString()
	case lexer.INTEGER:
		return p.parseInteger()
	case lexer.FLOAT:
		return p.parseFloat()
	case lexer.OPEN_BRACKET:
		return p.parseArrayLiteral()
	case lexer.OPEN_BRACE:
		return p.parseBlock()
	case lexer.OPEN_PAREN:
		return p.parseParenExpression()
	case lexer.COMMENT:
		p.Consume()
		return Node{Type: ILLEGAL}
	default:
		p.Consume()
		return Node{Type: ILLEGAL}
	}
}

func (p *Parser) parsePostfix(left Node) Node {
	for !p.AtEnd() {
		if p.match(lexer.OPEN_BRACKET) {
			index := p.parseExpression()
			p.Expect(lexer.CLOSED_BRACKET)
			left = NewToken("index", INDEX, left.Line, []Node{left, index})
			continue
		}

		if p.Current().Type == lexer.DOT && p.Peek(1).Type == lexer.IDENTIFIER {
			p.Consume()
			name := p.Current()
			p.Expect(lexer.IDENTIFIER)

			right := NewToken(name.Name, IDENTIFIER, name.Line, []Node{})
			left = NewToken(name.Name, NAMESPACE, name.Line, []Node{left, right})
			continue
		}

		if p.match(lexer.OPEN_PAREN) {
			left = Node{
				Name:     left.Name,
				Type:     FUNCTION_CALL,
				Line:     left.Line,
				Children: append([]Node{left}, p.parseArguments()...),
			}
			continue
		}

		break
	}

	return left
}

func (p *Parser) parseExpression() Node {
	return p.parsePostfix(p.parsePrimary())
}

func (p *Parser) parseAssignment(name Node) Node {
	assign := p.Consume()
	value := p.parseExpression()
	children := []Node{name, value}

	if !p.AtEnd() && p.Current().Type == lexer.ASSIGN {
		if p.Current().Name == "=>" {
			arrow := p.Consume()
			body := p.parseExpression()
			children = append(children, NewToken(arrow.Name, FUNCTION_DECL, arrow.Line, []Node{body}))
		} else {
			children[1] = p.parseAssignment(value)
		}
	}

	return NewToken(assign.Name, ASSIGNMENT, assign.Line, children)
}

func (p *Parser) parseStatement() Node {
	if p.Current().Type == lexer.COMMENT {
		p.Consume()
		return Node{Type: ILLEGAL}
	}

	node := p.parseExpression()

	if !p.AtEnd() && p.Current().Type == lexer.ASSIGN {
		return p.parseAssignment(node)
	}

	return node
}

func (p *Parser) Parse() {
	p.Output = []Node{}

	for !p.AtEnd() {
		start := p.Pos
		node := p.parseStatement()
		if node.Type != ILLEGAL {
			p.Output = append(p.Output, node)
		}

		if p.Pos == start {
			p.Consume()
		}
	}
}
