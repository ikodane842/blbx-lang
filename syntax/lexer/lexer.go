package lexer

import "unicode"

func IsAlphanumeric(s rune) bool {
	if !unicode.IsLetter(s) && !unicode.IsDigit(s) {
		return false
	}
	return true
}

func IsDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

type Lexer struct {
	Input  []rune
	Output []LexerToken
	Pos    int
}

func (l *Lexer) Set(s string) {
	l.Input = []rune(s)
}

func (l *Lexer) Get() []LexerToken {
	return l.Output
}

func (l *Lexer) Consume() rune {
	ch := l.Input[l.Pos]
	l.Pos++
	return ch
}

func (l *Lexer) AddToken(_name string, _type TokenType, _line int) {
	new_token := NewToken(_name, _type, _line)
	l.Output = append(l.Output, new_token)
}

func (l *Lexer) Current() rune {
	return l.Input[l.Pos]
}

func (l *Lexer) Peek() rune {
	if l.Pos+1 >= len(l.Input) {
		return 0
	}
	return l.Input[l.Pos+1]
}

func (l *Lexer) AtEnd() bool {
	return l.Pos >= len(l.Input)
}

func (l *Lexer) readString(line int) {
	// skip opening quote
	l.Consume()

	start := l.Pos

	for !l.AtEnd() && l.Current() != '"' {
		l.Consume()
	}

	// slice the runes between quotes
	value := string(l.Input[start:l.Pos])

	// skip closing quote
	l.Consume()

	l.AddToken(value, STRING, line)
}

func (l *Lexer) readNamespace(line int) {
	value := ""
	start := l.Pos

	for !l.AtEnd() && (IsAlphanumeric(l.Current()) || l.Current() == '_') {
		l.Consume()
	}

	value = string(l.Input[start:l.Pos])

	l.AddToken(value, IDENTIFIER, line)
}

func (l *Lexer) readNumber(line int) {
	start := l.Pos
	isFloat := false

	// read leading digits
	for !l.AtEnd() && IsDigit(l.Current()) {
		l.Consume()
	}

	// check for decimal part
	if !l.AtEnd() && l.Current() == '.' && IsDigit(l.Peek()) {
		isFloat = true
		l.Consume() // consume '.'

		for !l.AtEnd() && IsDigit(l.Current()) {
			l.Consume()
		}
	}

	value := string(l.Input[start:l.Pos])

	if isFloat {
		l.AddToken(value, FLOAT, line)
	} else {
		l.AddToken(value, INTEGER, line)
	}
}

func (l *Lexer) readSingleLineComment(line int) {
	// skip //
	l.Consume()
	l.Consume()

	start := l.Pos

	for !l.AtEnd() && l.Current() != '\n' {
		l.Consume()
	}

	value := string(l.Input[start:l.Pos])

	l.AddToken(value, COMMENT, line)
}

func (l *Lexer) readMultiLineComment(line *int) {
	// skip /*
	l.Consume()
	l.Consume()

	start := l.Pos

	for !l.AtEnd() {
		if l.Current() == '\n' {
			(*line)++
		}

		// detect */
		if l.Current() == '*' && l.Peek() == '/' {
			value := string(l.Input[start:l.Pos])

			// consume */
			l.Consume()
			l.Consume()

			l.AddToken(value, COMMENT, *line)
			return
		}

		l.Consume()
	}
}

func (l *Lexer) Tokenize() {
	line := 1

	for !l.AtEnd() {
		ch := l.Current()

		// handle whitespace
		if ch == ' ' || ch == '\t' {
			l.Consume()
			continue
		}

		// handle new-line
		if ch == '\n' {
			line++
			l.Consume()
			continue
		}

		// handle string
		if ch == '"' {
			l.readString(line)
			continue
		}

		// handle namespace
		if IsAlphanumeric(l.Current()) {
			l.readNamespace(line)
			continue
		}

		// handle assign
		if l.Current() == '=' {
			l.Consume()
			l.AddToken("=", ASSIGN, line)
			continue
		}

		if l.Current() == ':' {
			l.Consume()
			l.AddToken(":", ASSIGN, line)
			continue
		}

		//handle dot
		if l.Current() == '.' {
			l.Consume()
			l.AddToken(".", DOT, line)
			continue
		}

		//handle comma
		if l.Current() == ',' {
			l.Consume()
			l.AddToken(",", COMMA, line)
			continue
		}

		// handle opened paren
		if l.Current() == '(' {
			l.Consume()
			l.AddToken("(", OPEN_PAREN, line)
			continue
		}

		// handle closed paren
		if l.Current() == ')' {
			l.Consume()
			l.AddToken(")", CLOSED_PAREN, line)
			continue
		}

		// handle opened bracket
		if l.Current() == '[' {
			l.Consume()
			l.AddToken("[", OPEN_BRACKET, line)
			continue
		}

		// handle closed paren
		if l.Current() == ']' {
			l.Consume()
			l.AddToken("]", CLOSED_BRACKET, line)
			continue
		}

		// handle opened brace
		if l.Current() == '{' {
			l.Consume()
			l.AddToken("{", OPEN_BRACE, line)
			continue
		}

		// handle closed brace
		if l.Current() == '}' {
			l.Consume()
			l.AddToken("}", CLOSED_BRACE, line)
			continue
		}

		// handle integer
		if IsDigit(l.Current()) {
			l.readNumber(line)
			continue
		}

		// handle slinge-line comment
		if l.Current() == '/' && l.Peek() == '/' {
			l.readSingleLineComment(line)
			continue
		}

		// handle multi-line comment
		if l.Current() == '/' && l.Peek() == '*' {
			l.readMultiLineComment(&line)
			continue
		}

		l.Consume()
	}
}
