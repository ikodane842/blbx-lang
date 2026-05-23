package parser

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
	ARRAY_LITERAL
	TUPLE_LITERAL

	// statements
	VARIABLE_DECL
	ASSIGNMENT
	RETURN_STATEMENT

	// blocks/scopes
	BLOCK
	FUNCTION_DECL
	CLASS_DECL

	// control flow
	IF_STATEMENT
	FOR_LOOP
	WHILE_LOOP
)
