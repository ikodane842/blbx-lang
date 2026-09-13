package ir

import (
	"blbx_lang/syntax/parser"
	"strings"
)

func Lower(nodes []parser.Node) Node {
	program := Node{
		Type:     Program,
		Children: []Node{},
	}

	for _, node := range nodes {
		lowered := lowerNode(node)
		if lowered.Type != Illegal {
			program.Children = append(program.Children, lowered)
		}
	}

	return program
}

func lowerNode(node parser.Node) Node {
	switch node.Type {
	case parser.INTEGER_LITERAL:
		return literal(node, "integer")
	case parser.FLOAT_LITERAL:
		return literal(node, "float")
	case parser.STRING_LITERAL:
		return literal(node, "string")
	case parser.BOOLEAN_LITERAL:
		return literal(node, "boolean")
	case parser.IDENTIFIER:
		return Node{Type: Identifier, Name: node.Name, Line: node.Line}
	case parser.NAMESPACE:
		return lowerChildren(node, Member)
	case parser.INDEX:
		return lowerChildren(node, Index)
	case parser.BINARY_EXPR:
		return lowerBinary(node)
	case parser.ARRAY_LITERAL:
		return lowerChildren(node, Array)
	case parser.TUPLE_LITERAL:
		return lowerChildren(node, Tuple)
	case parser.BLOCK:
		return lowerBlock(node)
	case parser.FUNCTION_CALL:
		return lowerCall(node)
	case parser.FUNCTION_DECL:
		return lowerFunction(node)
	case parser.RETURN_STATEMENT:
		return lowerChildren(node, Return)
	case parser.ASSERT_STATEMENT:
		return lowerChildren(node, Assert)
	case parser.IMPORT_STATEMENT:
		return lowerImport(node)
	case parser.FROM_IMPORT_STATEMENT:
		return lowerFromImport(node)
	case parser.ASSIGNMENT:
		return lowerAssignment(node)
	default:
		return Node{Type: Illegal, Line: node.Line}
	}
}

func literal(node parser.Node, dataType string) Node {
	return Node{
		Type:     Literal,
		Value:    node.Name,
		DataType: dataType,
		Line:     node.Line,
	}
}

func lowerChildren(node parser.Node, nodeType Type) Node {
	lowered := Node{
		Type:     nodeType,
		Name:     node.Name,
		Line:     node.Line,
		Children: []Node{},
	}

	for _, child := range node.Children {
		loweredChild := lowerNode(child)
		if loweredChild.Type != Illegal {
			lowered.Children = append(lowered.Children, loweredChild)
		}
	}

	return lowered
}

func lowerBinary(node parser.Node) Node {
	binary := lowerChildren(node, Binary)
	binary.Name = node.Name
	return binary
}

func lowerBlock(node parser.Node) Node {
	block := lowerChildren(node, Block)
	block.Name = ""
	return block
}

func lowerFunction(node parser.Node) Node {
	function := lowerChildren(node, Function)
	function.Name = ""
	return function
}

func lowerCall(node parser.Node) Node {
	callType := Call

	switch node.Name {
	case "assert":
		callType = Assert
	case "if":
		callType = If
	case "for":
		callType = For
	case "while":
		callType = While
	}

	call := lowerChildren(node, callType)
	call.Name = node.Name
	return call
}

func lowerImport(node parser.Node) Node {
	importNode := Node{
		Type:     Import,
		Name:     node.Name,
		DataType: "import",
		Line:     node.Line,
	}

	pathLength := 0
	if node.Name != "" {
		pathLength = len(strings.Split(node.Name, "."))
	}

	if len(node.Children) > pathLength {
		importNode.Value = node.Children[len(node.Children)-1].Name
	}

	return importNode
}

func lowerFromImport(node parser.Node) Node {
	importNode := Node{
		Type:     Import,
		Name:     node.Name,
		DataType: "from",
		Line:     node.Line,
		Children: []Node{},
	}

	for _, child := range node.Children {
		imported := Node{
			Type: Identifier,
			Name: child.Name,
			Line: child.Line,
		}

		if len(child.Children) > 0 {
			imported.Value = child.Children[0].Name
		}

		importNode.Children = append(importNode.Children, imported)
	}

	return importNode
}

func lowerAssignment(node parser.Node) Node {
	if len(node.Children) == 0 {
		return Node{Type: Illegal, Line: node.Line}
	}

	if isFunctionAssignment(node) {
		return lowerFunctionAssignment(node)
	}

	target := lowerNode(node.Children[0])
	assign := Node{
		Type:     Assign,
		Line:     node.Line,
		Children: []Node{target},
	}

	if target.Type == Identifier {
		assign.Name = target.Name
	}

	for _, child := range node.Children[1:] {
		loweredChild := lowerNode(child)
		if loweredChild.Type != Illegal {
			assign.Children = append(assign.Children, loweredChild)
		}
	}

	return assign
}

func isFunctionAssignment(node parser.Node) bool {
	if len(node.Children) < 3 {
		return false
	}

	return node.Children[len(node.Children)-1].Type == parser.FUNCTION_DECL
}

func lowerFunctionAssignment(node parser.Node) Node {
	target := lowerNode(node.Children[0])
	function := Node{
		Type:     Function,
		Line:     node.Line,
		Children: []Node{},
	}

	if target.Type == Identifier {
		function.Name = target.Name
	}

	for _, child := range node.Children[1:] {
		if child.Type == parser.FUNCTION_DECL {
			for _, fnChild := range child.Children {
				loweredChild := lowerNode(fnChild)
				if loweredChild.Type != Illegal {
					function.Children = append(function.Children, loweredChild)
				}
			}
			continue
		}

		loweredChild := lowerNode(child)
		if loweredChild.Type != Illegal {
			function.Children = append(function.Children, loweredChild)
		}
	}

	return Node{
		Type:     Assign,
		Name:     function.Name,
		Line:     node.Line,
		Children: []Node{target, function},
	}
}
