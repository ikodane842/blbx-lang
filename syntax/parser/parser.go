package parser

import (
	"blbx_lang/syntax/diagnostic"
	"blbx_lang/syntax/lexer"
	"fmt"
	"strings"
)

type Parser struct {
	Input       []lexer.LexerToken
	Output      []Node
	Pos         int
	Diagnostics []diagnostic.Diagnostic
	eof         lexer.LexerToken
	depth       int
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
		return p.eof
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
	*p = Parser{Output: []Node{}, eof: lexer.LexerToken{Type: lexer.EOF, Line: 1, Column: 1, EndLine: 1, EndColumn: 1}}
	for _, tok := range tokens {
		if tok.Type == lexer.EOF {
			p.eof = tok
			continue
		}
		// Comments are trivia everywhere, including between an operator and value.
		if tok.Type != lexer.COMMENT {
			p.Input = append(p.Input, tok)
		}
		p.eof.Line, p.eof.Column = tok.EndLine, tok.EndColumn
		p.eof.EndLine, p.eof.EndColumn = tok.EndLine, tok.EndColumn
	}
}

func (p *Parser) Expect(tt lexer.TokenType) {
	if p.AtEnd() || p.Current().Type != tt {
		p.fail(diagnostic.UnexpectedToken, fmt.Sprintf("expected %v, got %s", tt, p.describe()))
	}

	p.Consume()
}

// syntaxFailure is recovered only at a statement boundary. Unexpected panics
// are deliberately not hidden as user syntax errors.
type syntaxFailure struct{}

func (p *Parser) describe() string {
	if p.AtEnd() {
		return "end of file"
	}
	return fmt.Sprintf("%q", p.Current().Name)
}

func (p *Parser) fail(code, message string) {
	p.report(code, message)
	panic(syntaxFailure{})
}

func (p *Parser) report(code, message string) {
	t := p.Current()
	p.Diagnostics = append(p.Diagnostics, diagnostic.Diagnostic{
		Line: t.Line, Column: t.Column, EndLine: t.EndLine, EndColumn: t.EndColumn,
		Severity: "error", Code: code, Message: message,
	})
}

func (p *Parser) enter() func() {
	if p.depth >= 256 {
		p.fail(diagnostic.NestingLimit, "syntax nesting exceeds limit of 256")
	}
	p.depth++
	return func() { p.depth-- }
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
		lexer.SELF,
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
	return p.isFunctionLiteralAt(p.Pos)
}

func (p *Parser) isFunctionLiteralAt(start int) bool {
	if start >= len(p.Input) || p.Input[start].Type != lexer.OPEN_PAREN {
		return false
	}

	depth := 0
	for index := start; index < len(p.Input); index++ {
		tok := p.Input[index]
		switch tok.Type {
		case lexer.OPEN_PAREN:
			depth++
		case lexer.CLOSED_PAREN:
			depth--
			if depth == 0 {
				if index+1 >= len(p.Input) {
					return false
				}
				next := p.Input[index+1]
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
		if !p.listSeparator(lexer.CLOSED_PAREN, "function arguments") {
			break
		}
	}

	p.Expect(lexer.CLOSED_PAREN)
	return args
}

// Newlines and comments do not replace commas in delimited lists. A comma
// before the closing delimiter is allowed, but empty entries are not.
func (p *Parser) listSeparator(end lexer.TokenType, context string) bool {
	if p.Current().Type == end {
		return false
	}
	if p.match(lexer.COMMA) {
		return true
	}
	message := fmt.Sprintf("expected ',' between %s, got %s", context, p.describe())
	if p.startsExpression() {
		p.report(diagnostic.MissingComma, message)
		return true
	}
	p.fail(diagnostic.MissingComma, message)
	return false
}

func (p *Parser) parseFunctionCall() Node {
	tok := p.Consume()
	p.Expect(lexer.OPEN_PAREN)

	return Node{
		DirectCall: true,
		Name:       tok.Name,
		Type:       FUNCTION_CALL,
		Line:       tok.Line,
		Children:   p.parseArguments(),
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
		p.fail(diagnostic.UnexpectedToken, "expected module path")
	}

	for !p.AtEnd() {
		tok := p.Current()
		p.Expect(lexer.IDENTIFIER)
		parts = append(parts, NewToken(tok.Name, IDENTIFIER, tok.Line, []Node{}))

		if p.Current().Type != lexer.DOT {
			break
		}

		p.Consume()
		if p.AtEnd() {
			p.fail(diagnostic.UnexpectedToken, "expected module name after '.'")
		}
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

	p.Expect(lexer.IMPORT)
	if p.Current().Type != lexer.IDENTIFIER {
		p.fail(diagnostic.UnexpectedToken, "expected imported name")
	}

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
		if p.Current().Type != lexer.IDENTIFIER {
			p.fail(diagnostic.UnexpectedToken, "expected imported name after ','")
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
		if !p.listSeparator(lexer.CLOSED_BRACKET, "array elements") {
			break
		}
	}

	p.Expect(lexer.CLOSED_BRACKET)
	return NewToken("array", ARRAY_LITERAL, tok.Line, items)
}

func (p *Parser) parseBlock() Node {
	tok := p.Consume()
	body := []Node{}
	object := p.Current().Type == lexer.STRING && p.Peek(1).Type == lexer.ASSIGN && p.Peek(1).Name == ":"

	for !p.AtEnd() && p.Current().Type != lexer.CLOSED_BRACE {
		p.skipComments()
		if p.AtEnd() || p.Current().Type == lexer.CLOSED_BRACE {
			break
		}

		start := p.Pos
		if object && !(p.Current().Type == lexer.STRING && p.Peek(1).Type == lexer.ASSIGN && p.Peek(1).Name == ":") {
			p.fail(diagnostic.ObjectField, "expected a quoted object key followed by ':'")
		}
		stmt := p.parseStatement()
		if stmt.Type != ILLEGAL {
			body = append(body, stmt)
		}
		if object {
			if !p.listSeparator(lexer.CLOSED_BRACE, "object fields") {
				break
			}
		} else if stmt.Type == ASSIGNMENT && len(stmt.Children) > 0 && stmt.Children[0].Type == STRING_LITERAL {
			p.fail(diagnostic.ObjectField, "object fields cannot be mixed with block statements")
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
		if !p.listSeparator(lexer.CLOSED_PAREN, "parenthesized items or parameters") {
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
		p.fail(diagnostic.ExpectedExpression, "expected expression, got end of file")
	}

	switch p.Current().Type {
	case lexer.SELF:
		return p.parseIdentifier()
	case lexer.ASSERT:
		return p.parseAssert()
	case lexer.IDENTIFIER:
		if p.Current().Name == "true" || p.Current().Name == "false" {
			return p.parseBoolean()
		}

		if p.Peek(1).Type == lexer.OPEN_PAREN && !p.isFunctionLiteralAt(p.Pos+1) {
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
		if p.isFunctionLiteralStart() {
			params := p.parseParenExpression()
			validateParams := []Node{params}
			if params.Type == TUPLE_LITERAL {
				validateParams = params.Children
			}
			for _, param := range validateParams {
				if param.Type == IDENTIFIER && param.Name == "self" {
					p.fail(diagnostic.SelfBinding, "self cannot be used as a parameter")
				}
			}
			arrow := p.Consume()
			body := p.parseExpression()
			return NewToken(arrow.Name, FUNCTION_DECL, arrow.Line, []Node{params, body})
		}
		return p.parseParenExpression()
	case lexer.COMMENT:
		p.Consume()
		return Node{Type: ILLEGAL}
	default:
		p.fail(diagnostic.ExpectedExpression, fmt.Sprintf("expected expression, got %s", p.describe()))
		return Node{Type: ILLEGAL}
	}
}

func (p *Parser) parsePostfix(left Node) Node {
	for !p.AtEnd() {
		if p.Pos > 0 && p.Current().Line > p.Input[p.Pos-1].EndLine && p.startsDestructuring() {
			break
		}
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
	defer p.enter()()
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
	defer p.enter()()
	if name.Type == IDENTIFIER && name.Name == "self" {
		p.fail(diagnostic.SelfBinding, "self cannot be reassigned or used as a parameter")
	}
	if p.Current().Name != "=>" && name.Type != IDENTIFIER && name.Type != NAMESPACE && name.Type != INDEX && !(name.Type == STRING_LITERAL && p.Current().Name == ":") {
		p.fail(diagnostic.InvalidTarget, "assignment target must be a name, member, or index")
	}
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
	if p.startsDestructuring() {
		target := p.parsePattern(map[string]bool{})
		assign := p.Current()
		p.Expect(lexer.ASSIGN)
		value := p.parseExpression()
		return NewToken(assign.Name, ASSIGNMENT, assign.Line, []Node{target, value})
	}
	if p.Current().Type == lexer.CLASS {
		return p.parseClass()
	}

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

func (p *Parser) parseClass() Node {
	defer p.enter()()
	tok := p.Consume()
	name := p.Current()
	p.Expect(lexer.IDENTIFIER)
	var base *Node
	if p.match(lexer.EXTENDS) {
		parent := p.Current()
		p.Expect(lexer.IDENTIFIER)
		value := NewToken(parent.Name, IDENTIFIER, parent.Line, nil)
		for p.match(lexer.DOT) {
			member := p.Current()
			p.Expect(lexer.IDENTIFIER)
			value = NewToken(member.Name, NAMESPACE, member.Line, []Node{value, NewToken(member.Name, IDENTIFIER, member.Line, nil)})
		}
		base = &value
	}
	if p.Current().Type != lexer.OPEN_BRACE {
		p.fail(diagnostic.UnexpectedToken, "expected class body '{'")
	}
	body := p.parseBlock()
	constructors := 0
	for _, member := range body.Children {
		if member.Type == FUNCTION_DECL {
			constructors++
			if constructors > 1 {
				p.fail(diagnostic.DuplicateConstructor, "class may contain only one unassigned anonymous constructor")
			}
		} else if member.Type != ASSIGNMENT || len(member.Children) == 0 || member.Children[0].Type != IDENTIFIER {
			p.fail(diagnostic.ClassSyntax, "class body requires named fields, methods, or an unassigned anonymous constructor")
		}
	}
	class := NewToken(name.Name, CLASS_DECL, tok.Line, body.Children)
	class.Base = base
	return class
}

func (p *Parser) Parse() {
	p.Output = []Node{}
	p.Diagnostics = nil

	for !p.AtEnd() {
		p.skipComments()
		if p.AtEnd() {
			break
		}

		start := p.Pos
		node := p.parseRecoveringStatement()
		if node.Type != ILLEGAL {
			p.Output = append(p.Output, node)
		}

		if p.Pos == start {
			p.Consume()
		}
	}
}

func (p *Parser) parseRecoveringStatement() (node Node) {
	start := p.Pos
	defer func() {
		if failure := recover(); failure != nil {
			if _, ok := failure.(syntaxFailure); !ok {
				panic(failure)
			}
			// Resume at the next source line; a malformed statement is discarded.
			line := p.Current().Line
			if p.Pos == start && !p.AtEnd() {
				p.Consume()
			}
			for !p.AtEnd() && p.Current().Line <= line {
				p.Consume()
			}
			node = Node{Type: ILLEGAL}
		}
	}()
	return p.parseStatement()
}
