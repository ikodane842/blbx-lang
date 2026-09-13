package parser

import (
	"blbx_lang/syntax/lexer"
	"log"
	"strings"
)

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
	if p.AtEnd() {
		log.Fatalf(
			"syntax error: expected %v, got EOF",
			tt,
		)
	}

	if p.Current().Type != tt {
		tok := p.Current()

		log.Fatalf(
			"[blbx][syntax]: error at line %d; expected %v, got %v (%q)",
			tok.Line,
			tt,
			tok.Type,
			tok.Name,
		)
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

func (p *Parser) skipComments() {
	for !p.AtEnd() && p.Current().Type == lexer.COMMENT {
		p.Consume()
	}
}

func (p *Parser) startsExpression() bool {
	if p.AtEnd() {
		return false
	}

	switch p.Current().Type {
	case lexer.ASSERT,
		lexer.IDENTIFIER,
		lexer.STRING,
		lexer.INTEGER,
		lexer.FLOAT,
		lexer.OPEN_BRACKET,
		lexer.OPEN_BRACE,
		lexer.OPEN_PAREN:
		return true
	default:
		return false
	}
}

func (p *Parser) isFunctionLiteralStart() bool {
	if p.AtEnd() || p.Current().Type != lexer.OPEN_PAREN {
		return false
	}

	depth := 0
	for offset := 0; p.Pos+offset < len(p.Input); offset++ {
		tok := p.Peek(offset)
		switch tok.Type {
		case lexer.OPEN_PAREN:
			depth++
		case lexer.CLOSED_PAREN:
			depth--
			if depth == 0 {
				next := p.Peek(offset + 1)
				return next.Type == lexer.ASSIGN && next.Name == "=>"
			}
		}
	}

	return false
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
		p.skipComments()
		if p.AtEnd() || p.Current().Type == lexer.CLOSED_PAREN {
			break
		}

		start := p.Pos
		arg := p.parseStatement()
		if arg.Type != ILLEGAL {
			args = append(args, arg)
		}

		if p.Pos == start {
			p.Consume()
		}

		p.skipComments()
		if !p.match(lexer.COMMA) {
			if p.startsExpression() {
				continue
			}

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

func (p *Parser) parseAssert() Node {
	tok := p.Consume()
	p.Expect(lexer.OPEN_PAREN)

	return NewToken(
		tok.Name,
		ASSERT_STATEMENT,
		tok.Line,
		p.parseArguments(),
	)
}

func (p *Parser) parseImportPath() ([]Node, string) {
	parts := []Node{}

	if p.Current().Type != lexer.IDENTIFIER {
		return parts, ""
	}

	for !p.AtEnd() {
		tok := p.Current()
		p.Expect(lexer.IDENTIFIER)
		parts = append(parts, NewToken(tok.Name, IDENTIFIER, tok.Line, []Node{}))

		if p.Current().Type != lexer.DOT {
			break
		}

		p.Consume()
	}

	names := []string{}
	for _, part := range parts {
		names = append(names, part.Name)
	}

	return parts, strings.Join(names, ".")
}

func (p *Parser) parseImportStatement() Node {
	tok := p.Consume()
	pathParts, path := p.parseImportPath()
	children := pathParts

	if p.Current().Type == lexer.AS {
		p.Consume()
		alias := p.Current()
		p.Expect(lexer.IDENTIFIER)
		children = append(children, NewToken(alias.Name, IDENTIFIER, alias.Line, []Node{}))
	}

	return NewToken(path, IMPORT_STATEMENT, tok.Line, children)
}

func (p *Parser) parseFromImportStatement() Node {
	tok := p.Consume()
	_, path := p.parseImportPath()

	if p.Current().Type != lexer.IMPORT {
		return NewToken(path, FROM_IMPORT_STATEMENT, tok.Line, []Node{})
	}

	p.Consume()

	imports := []Node{}
	for !p.AtEnd() && p.Current().Type == lexer.IDENTIFIER {
		name := p.Current()
		p.Expect(lexer.IDENTIFIER)
		children := []Node{}

		if p.Current().Type == lexer.AS {
			p.Consume()
			alias := p.Current()
			p.Expect(lexer.IDENTIFIER)
			children = append(children, NewToken(alias.Name, IDENTIFIER, alias.Line, []Node{}))
		}

		imports = append(imports, NewToken(name.Name, IDENTIFIER, name.Line, children))

		if !p.match(lexer.COMMA) {
			break
		}
	}

	return NewToken(path, FROM_IMPORT_STATEMENT, tok.Line, imports)
}

func (p *Parser) parseArrayLiteral() Node {
	tok := p.Consume()
	items := []Node{}

	for !p.AtEnd() && p.Current().Type != lexer.CLOSED_BRACKET {
		p.skipComments()
		if p.AtEnd() || p.Current().Type == lexer.CLOSED_BRACKET {
			break
		}

		start := p.Pos
		item := p.parseExpression()
		if item.Type != ILLEGAL {
			items = append(items, item)
		}

		if p.Pos == start {
			p.Consume()
		}

		p.skipComments()
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
		p.skipComments()
		if p.AtEnd() || p.Current().Type == lexer.CLOSED_BRACE {
			break
		}

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
		p.skipComments()
		if p.AtEnd() || p.Current().Type == lexer.CLOSED_PAREN {
			break
		}

		start := p.Pos
		item := p.parseStatement()
		if item.Type != ILLEGAL {
			items = append(items, item)
		}

		if p.Pos == start {
			p.Consume()
		}

		p.skipComments()
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
	case lexer.ASSERT:
		return p.parseAssert()
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

		if p.Current().Type == lexer.OPEN_PAREN {
			if p.isFunctionLiteralStart() {
				break
			}
			p.Consume()
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
	return p.parseBinaryExpression()
}

func (p *Parser) parseBinaryExpression() Node {
	left := p.parsePostfix(p.parsePrimary())

	for !p.AtEnd() && p.Current().Type == lexer.STAR {
		operator := p.Consume()
		right := p.parsePostfix(p.parsePrimary())
		left = NewToken(operator.Name, BINARY_EXPR, operator.Line, []Node{left, right})
	}

	return left
}

func (p *Parser) parseAssignment(name Node) Node {
	assign := p.Consume()
	value := p.parseExpression()

	if assign.Name == "=>" {
		return NewToken(assign.Name, FUNCTION_DECL, assign.Line, []Node{name, value})
	}

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
	p.skipComments()

	//handling comments
	if p.Current().Type == lexer.COMMENT {
		p.Consume()
		return Node{Type: ILLEGAL}
	}

	if p.Current().Type == lexer.RETURN {
		tok := p.Consume()
		value := p.parseExpression()
		return NewToken(tok.Name, RETURN_STATEMENT, tok.Line, []Node{value})
	}

	if p.Current().Type == lexer.IMPORT {
		return p.parseImportStatement()
	}

	if p.Current().Type == lexer.FROM {
		return p.parseFromImportStatement()
	}

	// a statement can both be a expression
	node := p.parseExpression()

	// or an assignment
	if !p.AtEnd() && p.Current().Type == lexer.ASSIGN {
		return p.parseAssignment(node)
	}

	return node
}

func (p *Parser) Parse() {
	p.Output = []Node{}

	for !p.AtEnd() {
		p.skipComments()
		if p.AtEnd() {
			break
		}

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
