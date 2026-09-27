package lexer

import "blbx_lang/syntax/diagnostic"

// Formatted strings retain expression tokens so the normal parser and checker
// handle names, calls, and source locations inside each interpolation.
func (l *Lexer) readFormattedString() {
	l.formatDepth++
	defer func() { l.formatDepth-- }()
	if l.formatDepth > 128 {
		l.report(diagnostic.NestingLimit, "formatted string nesting limit exceeded")
		l.Pos = len(l.Input)
		return
	}
	l.Consume() // f
	l.Consume() // opening quote
	l.AddToken("f\"", FSTRING_START, 0)
	l.start = l.Pos
	value := []rune{}
	flush := func() {
		l.AddToken(string(value), STRING, 0)
		value = nil
	}
	for !l.AtEnd() {
		switch l.Current() {
		case '"':
			flush()
			l.start = l.Pos
			l.Consume()
			l.AddToken("\"", FSTRING_END, 0)
			return
		case '{', '}':
			brace := l.Current()
			if l.Peek() == brace {
				l.Consume()
				l.Consume()
				value = append(value, brace)
				continue
			}
			flush()
			l.start = l.Pos
			l.Consume()
			if brace == '}' {
				l.report(diagnostic.UnexpectedToken, "literal closing brace in formatted string must be escaped as }}")
			} else {
				l.AddToken("{", INTERPOLATION_START, 0)
				l.tokenize(true)
			}
			l.start = l.Pos
		case '\\':
			l.Consume()
			if l.AtEnd() {
				break
			}
			escaped := l.Consume()
			switch escaped {
			case 'n':
				escaped = '\n'
			case 'r':
				escaped = '\r'
			case 't':
				escaped = '\t'
			case 'b':
				escaped = '\b'
			case 'f':
				escaped = '\f'
			}
			value = append(value, escaped)
		default:
			value = append(value, l.Consume())
		}
	}
	l.report(diagnostic.UnclosedString, "unterminated formatted string; expected closing quote")
}
