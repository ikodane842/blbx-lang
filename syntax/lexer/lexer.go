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

func (l *Lexer) Tokenize() {

}
