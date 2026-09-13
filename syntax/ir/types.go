package ir

type Type string

const (
	Program    Type = "PROGRAM"
	Literal    Type = "LITERAL"
	Identifier Type = "IDENTIFIER"
	Assign     Type = "ASSIGN"
	Block      Type = "BLOCK"
	Call       Type = "CALL"
	Member     Type = "MEMBER"
	Index      Type = "INDEX"
	Binary     Type = "BINARY"
	Array      Type = "ARRAY"
	Tuple      Type = "TUPLE"
	Function   Type = "FUNCTION"
	Return     Type = "RETURN"
	Assert     Type = "ASSERT"
	If         Type = "IF"
	For        Type = "FOR"
	While      Type = "WHILE"
	Illegal    Type = "ILLEGAL"
)

type Node struct {
	Type     Type   `json:"type"`
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	DataType string `json:"dataType,omitempty"`
	Line     int    `json:"line,omitempty"`
	Children []Node `json:"children,omitempty"`
}
