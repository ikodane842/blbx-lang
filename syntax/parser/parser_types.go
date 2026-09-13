package parser

import "encoding/json"

type ParserType int

const (
	ILLEGAL ParserType = iota

	// literals
	INTEGER_LITERAL
	FLOAT_LITERAL
	STRING_LITERAL
	BOOLEAN_LITERAL

	// identifiers
	IDENTIFIER
	NAMESPACE

	// expressions
	FUNCTION_CALL
	INDEX
	BINARY_EXPR
	ARRAY_LITERAL
	TUPLE_LITERAL

	// statements
	VARIABLE_DECL
	ASSIGNMENT
	RETURN_STATEMENT
	ASSERT_STATEMENT

	// blocks/scopes
	BLOCK
	FUNCTION_DECL
	CLASS_DECL

	// control flow
	IF_STATEMENT
	FOR_LOOP
	WHILE_LOOP
)

func (pt ParserType) String() string {
	switch pt {
	case INTEGER_LITERAL:
		return "INTEGER_LITERAL"
	case FLOAT_LITERAL:
		return "FLOAT_LITERAL"
	case STRING_LITERAL:
		return "STRING_LITERAL"
	case BOOLEAN_LITERAL:
		return "BOOLEAN_LITERAL"
	case IDENTIFIER:
		return "IDENTIFIER"
	case NAMESPACE:
		return "NAMESPACE"
	case FUNCTION_CALL:
		return "FUNCTION_CALL"
	case INDEX:
		return "INDEX"
	case BINARY_EXPR:
		return "BINARY_EXPR"
	case ARRAY_LITERAL:
		return "ARRAY_LITERAL"
	case TUPLE_LITERAL:
		return "TUPLE_LITERAL"
	case VARIABLE_DECL:
		return "VARIABLE_DECL"
	case ASSIGNMENT:
		return "ASSIGNMENT"
	case RETURN_STATEMENT:
		return "RETURN_STATEMENT"
	case ASSERT_STATEMENT:
		return "ASSERT_STATEMENT"
	case BLOCK:
		return "BLOCK"
	case FUNCTION_DECL:
		return "FUNCTION_DECL"
	case CLASS_DECL:
		return "CLASS_DECL"
	case IF_STATEMENT:
		return "IF_STATEMENT"
	case FOR_LOOP:
		return "FOR_LOOP"
	case WHILE_LOOP:
		return "WHILE_LOOP"
	default:
		return "ILLEGAL"
	}
}

func (pt ParserType) MarshalJSON() ([]byte, error) {
	return json.Marshal(pt.String())
}
