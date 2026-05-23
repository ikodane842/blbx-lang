package parser

type Node struct {
	Name     string
	Type     ParserType
	Line     int
	Children []Node
}

func NewToken(_name string, _type ParserType, _line int) Node {
	return Node{
		Name: _name,
		Type: _type,
		Line: _line,
		Children: []Node
	}
}
