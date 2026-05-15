package lexer

type LexerToken struct {
	Name string
	Type TokenType
	Line int
}

func NewToken(_name string, _type TokenType, _line int) LexerToken {
	return LexerToken{
		Name: _name,
		Type: _type,
		Line: _line,
	}
}
