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

func (p *Parser) Set(tokens []lexer.LexerToken) {
	p.Input = tokens
	p.Pos = 0
}

func (p *Parser) Expect(tt lexer.TokenType)

func (p *Parser) parseIdentifier() Node {
	tok := p.Consume()

	return Node{
		Type:  IDENTIFIER,
		Value: tok.Name,
	}
}

func (p *Parser) parseFunctionCall() Node {

	// function name
	name := p.Consume()

	// consume '('
	p.Consume()

	args := []Node{}

	for !p.AtEnd() && p.Current().Type != lexer.CLOSED_PAREN {

		arg := p.parseExpression()

		args = append(args, arg)

		if p.Current().Type == lexer.COMMA {
			p.Consume()
		}
	}

	// consume ')'
	p.Consume()

	return Node{
		Type:     FUNCTION_CALL,
		Value:    name.Name,
		Children: args,
	}
}

func (p *Parser) parseExpression() Node {

	// function call
	if p.Current().Type == lexer.IDENTIFIER &&
		p.Peek(1).Type == lexer.OPEN_PAREN {

		return p.parseFunctionCall()
	}

	// identifier
	if p.Current().Type == lexer.IDENTIFIER {
		return p.parseIdentifier()
	}

	// string
	if p.Current().Type == lexer.STRING {
		return p.parseString()
	}

	// integer
	if p.Current().Type == lexer.INTEGER {
		return p.parseInteger()
	}

	// fallback
	p.Consume()

	return Node{}
}

func (p *Parser) Parse() {

	for !p.AtEnd() {
		if p.Current().Type == lexer.IDENTIFIER {
			if p.Peek(1).Type == DOT {
				p.Consume()
				p.consume()

			}
		}
	}
}
