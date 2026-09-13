package interpreter

import (
	"blbx_lang/syntax/ir"
	"testing"
)

func TestForLoopCursorMethods(t *testing.T) {
	program := ir.Node{
		Type: ir.Program,
		Children: []ir.Node{
			forProgram([]ir.Node{
				printCall(methodCall("__", "elem")),
				printCall(methodCall("__", "idx")),
				printCall(methodCall("__", "step")),
			}),
		},
	}

	runtime := New()
	result, err := runtime.Execute(program)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	want := []string{"stand", "0", "1", "up", "1", "2", "and", "2", "3", "fight", "3", "4"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("output = %#v, want %#v", result.Output, want)
	}
}

func TestForLoopContinueAndBreak(t *testing.T) {
	program := ir.Node{
		Type: ir.Program,
		Children: []ir.Node{
			forProgram([]ir.Node{
				{
					Type: ir.If,
					Name: "if",
					Children: []ir.Node{
						methodCall("__", "elemEqAnd"),
						closureExpr([]ir.Node{methodCall("__", "break")}),
						closureExpr(nil),
					},
				},
				{
					Type: ir.If,
					Name: "if",
					Children: []ir.Node{
						methodCall("__", "elemEqUp"),
						closureExpr([]ir.Node{methodCall("__", "continue")}),
						closureExpr([]ir.Node{printCall(methodCall("__", "elem"))}),
					},
				},
			}),
		},
	}

	runtime := New()
	result, err := runtime.Execute(program)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	want := []string{"stand"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("output = %#v, want %#v", result.Output, want)
	}
}

func forProgram(body []ir.Node) ir.Node {
	return ir.Node{
		Type: ir.For,
		Name: "for",
		Children: []ir.Node{
			{
				Type: ir.Array,
				Children: []ir.Node{
					stringLiteral("stand"),
					stringLiteral("up"),
					stringLiteral("and"),
					stringLiteral("fight"),
				},
			},
			functionExpr("__", []ir.Node{
				{
					Type:     ir.Block,
					Children: body,
				},
			}),
		},
	}
}

func functionExpr(param string, children []ir.Node) ir.Node {
	fn := ir.Node{
		Type: ir.Function,
	}

	if param != "" {
		fn.Children = append(fn.Children, ir.Node{Type: ir.Identifier, Name: param})
	}

	if len(children) == 1 && children[0].Type == ir.Block {
		fn.Children = append(fn.Children, children[0])
		return fn
	}

	fn.Children = append(fn.Children, ir.Node{Type: ir.Block, Children: children})
	return fn
}

func closureExpr(children []ir.Node) ir.Node {
	return functionExpr("", children)
}

func printCall(arg ir.Node) ir.Node {
	return ir.Node{
		Type:     ir.Call,
		Name:     "print",
		Children: []ir.Node{arg},
	}
}

func methodCall(receiver string, method string) ir.Node {
	switch method {
	case "elemEqUp":
		return receiverMethodCall(methodCall(receiver, "elem"), "eq", []ir.Node{stringLiteral("up")})
	case "elemEqAnd":
		return receiverMethodCall(methodCall(receiver, "elem"), "eq", []ir.Node{stringLiteral("and")})
	}

	return receiverMethodCall(ir.Node{Type: ir.Identifier, Name: receiver}, method, nil)
}

func receiverMethodCall(receiver ir.Node, method string, args []ir.Node) ir.Node {
	return ir.Node{
		Type: ir.Call,
		Name: method,
		Children: append([]ir.Node{
			{
				Type: ir.Member,
				Name: method,
				Children: []ir.Node{
					receiver,
					{Type: ir.Identifier, Name: method},
				},
			},
		}, args...),
	}
}

func stringLiteral(value string) ir.Node {
	return ir.Node{
		Type:     ir.Literal,
		Value:    value,
		DataType: "string",
	}
}

func sameStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
