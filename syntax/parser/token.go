package parser

type Node struct {
	Bases      []Node `json:"bases,omitempty"`
	Interfaces []Node `json:"interfaces,omitempty"`
	Base       *Node  `json:"base,omitempty"`
	DirectCall bool   `json:"directCall,omitempty"`
	Name       string
	Type       ParserType
	Line       int
	Children   []Node
}

func NewToken(_name string, _type ParserType, _line int, _children []Node) Node {
	return Node{
		Name:     _name,
		Type:     _type,
		Line:     _line,
		Children: _children,
	}
}
