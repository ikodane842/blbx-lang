package lexer

type TokenType int

const (
	ILLEGAL TokenType = iota

	BOOLEAN
	INTEGER
	FLOAT
	STRING

	RETURN
	ASSERT
	IMPORT
	FROM
	AS
	IDENTIFIER
	ASSIGN

	COMMENT
	NEW_LINE

	OPEN_PAREN
	CLOSED_PAREN
	OPEN_BRACKET
	CLOSED_BRACKET
	OPEN_BRACE
	CLOSED_BRACE

	COMMA
	DOT
	STAR
)

func (tt TokenType) String() string {
	switch tt {
	case BOOLEAN:
		return "BOOLEAN"
	case INTEGER:
		return "INTEGER"
	case FLOAT:
		return "FLOAT"
	case STRING:
		return "STRING"
	case RETURN:
		return "RETURN"
	case ASSERT:
		return "ASSERT"
	case IMPORT:
		return "IMPORT"
	case FROM:
		return "FROM"
	case AS:
		return "AS"
	case IDENTIFIER:
		return "IDENTIFIER"
	case ASSIGN:
		return "ASSIGN"
	case COMMENT:
		return "COMMENT"
	case NEW_LINE:
		return "NEW_LINE"
	case OPEN_PAREN:
		return "OPEN_PAREN"
	case CLOSED_PAREN:
		return "CLOSED_PAREN"
	case OPEN_BRACKET:
		return "OPEN_BRACKET"
	case CLOSED_BRACKET:
		return "CLOSED_BRACKET"
	case OPEN_BRACE:
		return "OPEN_BRACE"
	case CLOSED_BRACE:
		return "CLOSED_BRACE"
	case COMMA:
		return "COMMA"
	case DOT:
		return "DOT"
	case STAR:
		return "STAR"
	default:
		return "ILLEGAL"
	}
}
