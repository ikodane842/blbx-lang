package lexer

type Lexer struct {
	input  []rune
	output []LexerToken
	pos    int
}

func (l *Lexer) Set(s string) {
	l.input = []rune(s)
}

func (l *Lexer) Consume() rune {
	ch := l.input[l.pos]
	l.pos++
	return ch
}

func (l *Lexer) AddToken(_name string, _type TokenType, _line int) {
	new_token := NewToken(_name, _type, _line)
	l.output = append(l.output, new_token)
}

func (l *Lexer) Current() rune {
	return l.input[l.pos]
}

func (l *Lexer) AtEnd() bool {
	return l.pos >= len(l.input)
}

func (l *Lexer) readString(line int) {
	// skip opening quote
	l.Consume()

	start := l.pos

	for !l.AtEnd() && l.Current() != '"' {
		l.Consume()
	}

	// slice the runes between quotes
	value := string(l.input[start:l.pos])

	// skip closing quote
	l.Consume()

	l.AddToken(value, STRING, line)
}

func (l *Lexer) Tokenize() {
	line := 1

	for !l.AtEnd() {
		ch := l.Current()

		// whitespace
		if ch == ' ' || ch == '\t' {
			l.Consume()
			continue
		}

		// newline
		if ch == '\n' {
			line++
			l.Consume()
			continue
		}

		// string
		if ch == '"' {
			l.readString(line)
			continue
		}

		// TODO: other tokens here

		l.Consume()
	}
}
