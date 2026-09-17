package ir

type Type string

const (
	Program       Type = "PROGRAM"
	Literal       Type = "LITERAL"
	Identifier    Type = "IDENTIFIER"
	Assign        Type = "ASSIGN"
	Block         Type = "BLOCK"
	Call          Type = "CALL"
	Member        Type = "MEMBER"
	Index         Type = "INDEX"
	Binary        Type = "BINARY"
	Array         Type = "ARRAY"
	Tuple         Type = "TUPLE"
	Function      Type = "FUNCTION"
	Class         Type = "CLASS"
	Return        Type = "RETURN"
	Assert        Type = "ASSERT"
	Import        Type = "IMPORT"
	If            Type = "IF"
	For           Type = "FOR"
	While         Type = "WHILE"
	Illegal       Type = "ILLEGAL"
	ArrayPattern  Type = "ARRAY_PATTERN"
	ObjectPattern Type = "OBJECT_PATTERN"
	PatternField  Type = "PATTERN_FIELD"
	RestPattern   Type = "REST_PATTERN"
)

type Node struct {
	Base     *Node  `json:"base,omitempty"`
	Type     Type   `json:"type"`
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	DataType string `json:"dataType,omitempty"`
	Line     int    `json:"line,omitempty"`
	Children []Node `json:"children,omitempty"`
}
