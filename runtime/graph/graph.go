// Package graph compiles BLBX IR into a graph of reusable expression components
// and ordered control regions. The VM never evaluates IR nodes.
package graph

import (
	"blbx_lang/syntax/ir"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

type Type string

const (
	ProgramRegion  Type = "PROGRAM"
	Literal        Type = "LITERAL"
	Identifier     Type = "IDENTIFIER"
	Assign         Type = "ASSIGN"
	Block          Type = "BLOCK"
	Call           Type = "CALL"
	MethodCall     Type = "METHOD_CALL"
	ExpressionCall Type = "EXPRESSION_CALL"
	Member         Type = "MEMBER"
	Index          Type = "INDEX"
	Binary         Type = "BINARY"
	Array          Type = "ARRAY"
	Tuple          Type = "TUPLE"
	Object         Type = "OBJECT"
	Function       Type = "FUNCTION"
	Class          Type = "CLASS"
	Interface      Type = "INTERFACE"
	Return         Type = "RETURN"
	Assert         Type = "ASSERT"
	Import         Type = "IMPORT"
	If             Type = "IF"
	For            Type = "FOR"
	While          Type = "WHILE"
	Illegal        Type = "ILLEGAL"
	ArrayPattern   Type = "ARRAY_PATTERN"
	ObjectPattern  Type = "OBJECT_PATTERN"
	PatternField   Type = "PATTERN_FIELD"
	RestPattern    Type = "REST_PATTERN"
	Step           Type = "STEP"
	LoopGate       Type = "LOOP_GATE"
	ValueRef       Type = "VALUE_REF"
	ArgumentRegion Type = "ARGUMENTS"
)

// Static is decoded once during graph construction. Literal visits merely read it.
type Static struct {
	Kind    string  `json:"kind"`
	Integer int64   `json:"integer,omitempty"`
	Float   float64 `json:"float,omitempty"`
	String  string  `json:"string,omitempty"`
	Boolean bool    `json:"boolean,omitempty"`
}

type Location struct {
	File string `json:"file,omitempty"`
	Line int    `json:"line"`
}

// Node is immutable once Build returns. Inputs are ordered operand/reference
// ports; Entry/Next are control/effect edges, not values to pre-evaluate.
// A node can have multiple incoming edges. Execution state belongs to the VM's
// activations, never to this shared graph (including in task workers).
type Node struct {
	ID                                  int
	Type                                Type
	Name, Value, DataType               string
	File                                string
	Line                                int
	Inputs                              []*Node
	Receiver, Callee                    *Node
	Arguments                           []*Node
	Method                              string
	Bases, Interfaces                   []*Node
	Base                                *Node
	Entry, Next                         *Node
	Condition, Then, Else, Body, Repeat *Node
	Iterable                            *Node
	Static                              *Static
	Uses                                []Location
	// Value nodes describe immutable scalar computations. Definition references
	// a completed binding version; Shared identifies an equivalent computation.
	Pure               bool
	ValueType          string
	Effect             string
	Definition, Shared *Node
	// Every region has an explicit effect dependency chain. Exit is its demand
	// root; EffectIn must complete before a step is activated.
	EffectIn, Exit *Node
	ArgumentPlan   *Node
}

type Program struct {
	Root  *Node
	Nodes []*Node
}

type builder struct {
	program  *Program
	file     string
	literals map[string]*Node
}

func Build(root ir.Node, file string) *Program {
	b := &builder{program: &Program{}, file: file, literals: map[string]*Node{}}
	b.program.Root = b.lower(root, false)
	b.program.separateValuesAndEffects()
	return b.program
}

func (b *builder) node(kind Type, line int) *Node {
	n := &Node{ID: len(b.program.Nodes) + 1, Type: kind, File: b.file, Line: line}
	b.program.Nodes = append(b.program.Nodes, n)
	return n
}

func (b *builder) lower(src ir.Node, assignmentValue bool) *Node {
	if src.Type == ir.Literal {
		key := src.DataType + "\x00" + src.Value
		if n := b.literals[key]; n != nil {
			n.Uses = append(n.Uses, Location{b.file, src.Line})
			return n
		}
		n := b.node(Literal, src.Line)
		n.DataType, n.Value = src.DataType, src.Value
		n.Static = &Static{Kind: src.DataType}
		switch src.DataType {
		case "integer":
			value, err := strconv.ParseInt(src.Value, 10, 64)
			if err == nil {
				n.Static.Integer = value
			}
		case "float":
			value, err := strconv.ParseFloat(src.Value, 64)
			if err == nil {
				n.Static.Float = value
			}
		case "boolean":
			n.Static.Boolean = src.Value == "true"
		default:
			n.Static.String = src.Value
		}
		n.Uses = []Location{{b.file, src.Line}}
		b.literals[key] = n
		return n
	}

	n := b.node(Type(src.Type), src.Line)
	n.Name, n.Value, n.DataType = src.Name, src.Value, src.DataType
	// The old IR represents object literals as blocks. Resolve that ambiguity
	// once, including the language's special empty-object assignment behavior.
	if src.Type == ir.Block && (objectLiteral(src) || (assignmentValue && len(src.Children) == 0)) {
		n.Type = Object
	}
	for index, child := range src.Children {
		isValue := src.Type == ir.Assign && index == len(src.Children)-1 && index > 0
		n.Inputs = append(n.Inputs, b.lower(child, isValue))
	}
	for _, base := range src.Bases {
		n.Bases = append(n.Bases, b.lower(base, false))
	}
	for _, contract := range src.Interfaces {
		n.Interfaces = append(n.Interfaces, b.lower(contract, false))
	}
	if src.Base != nil {
		n.Base = b.lower(*src.Base, false)
	}

	switch n.Type {
	case Call:
		if n.DataType != "direct-call" && len(n.Inputs) > 0 {
			callee := n.Inputs[0]
			if callee.Type == Member && len(callee.Inputs) >= 2 {
				n.Type = MethodCall
				n.Receiver, n.Method = callee.Inputs[0], callee.Inputs[1].Name
				n.Arguments = n.Inputs[1:]
			} else if callee.Type == Index {
				n.Type = ExpressionCall
				n.Callee, n.Arguments = callee, n.Inputs[1:]
			}
		}
	case ProgramRegion, Block:
		// Unique step nodes keep a shared literal/expression independent from the
		// continuation at each use site. Even discarded values remain reachable.
		var previous *Node
		for index, input := range n.Inputs {
			step := b.node(Step, src.Children[index].Line)
			step.Inputs = []*Node{input}
			if previous == nil {
				n.Entry = step
			} else {
				previous.Next = step
			}
			previous = step
		}
	case If:
		if len(n.Inputs) > 0 {
			n.Condition = n.Inputs[0]
		}
		if len(n.Inputs) > 1 {
			n.Then = n.Inputs[1]
		}
		if len(n.Inputs) > 2 {
			n.Else = n.Inputs[2]
		}
	case While, For:
		if len(n.Inputs) > 0 {
			n.Condition = n.Inputs[0]
		}
		if len(n.Inputs) > 1 {
			n.Body = n.Inputs[1]
		}
		gate := b.node(LoopGate, n.Line)
		gate.Name = "condition"
		if n.Type == For {
			// Evaluate the iterable once at loop entry. Repeated gate visits
			// request next() from the activation's iterator, not this expression.
			n.Iterable, n.Condition = n.Condition, nil
			gate.Name = "iterator-next"
		}
		gate.Condition, gate.Body = n.Condition, n.Body
		gate.Repeat = gate
		n.Entry = gate
	case Function:
		for _, input := range n.Inputs {
			if input.Type == Block || input.Type == Object {
				n.Body = input
			}
		}
	}
	return n
}

func objectLiteral(n ir.Node) bool {
	if len(n.Children) == 0 {
		return false
	}
	for _, field := range n.Children {
		if field.Type != ir.Assign || len(field.Children) < 2 || field.Children[0].Type != ir.Literal || field.Children[0].DataType != "string" {
			return false
		}
	}
	return true
}

// Edge is a serializable directed connection. Reference ports (e.g. a function
// body or assignment target) must not be eagerly visited as value dependencies.
type Edge struct {
	Port string `json:"port"`
	Kind string `json:"kind"`
	To   int    `json:"to"`
}

func (n *Node) Edges() []Edge {
	edges := []Edge{}
	add := func(port, kind string, target *Node) {
		if target != nil {
			edges = append(edges, Edge{port, kind, target.ID})
		}
	}
	for index, input := range n.Inputs {
		kind := "value"
		switch n.Type {
		case ProgramRegion, Block, Class, Interface, Function, Object, ArrayPattern, ObjectPattern, PatternField, RestPattern, Import:
			kind = "reference"
		case Assign:
			if index == 0 {
				kind = "reference"
			}
		case If, While, For:
			kind = "control"
		case Member:
			if index > 0 {
				kind = "reference"
			}
		case Step:
			kind = "control"
		case MethodCall, ExpressionCall:
			if index == 0 {
				kind = "reference"
			}
		}
		add(fmt.Sprintf("input:%d", index), kind, input)
	}
	for index, base := range n.Bases {
		add(fmt.Sprintf("base:%d", index), "reference", base)
	}
	for index, contract := range n.Interfaces {
		add(fmt.Sprintf("interface:%d", index), "reference", contract)
	}
	add("base", "reference", n.Base)
	add("receiver", "value", n.Receiver)
	add("callee", "value", n.Callee)
	add("entry", "control", n.Entry)
	// Forward navigation is not an effect prerequisite: marking both directions
	// as effects would turn every sequence into a false dependency cycle.
	add("next", "reference", n.Next)
	add("condition", "control", n.Condition)
	add("iterable", "value", n.Iterable)
	add("then", "control", n.Then)
	add("else", "control", n.Else)
	add("body", "reference", n.Body)
	add("repeat", "control", n.Repeat)
	add("definition", "value", n.Definition)
	add("shared-value", "value", n.Shared)
	add("effect-in", "effect", n.EffectIn)
	add("exit", "control", n.Exit)
	add("arguments", "control", n.ArgumentPlan)
	return edges
}

// WriteJSON flattens references to IDs, including cycles. It never executes code.
func (p *Program) WriteJSON(w io.Writer) error {
	type record struct {
		ID        int        `json:"id"`
		Operation Type       `json:"operation"`
		Name      string     `json:"name,omitempty"`
		Method    string     `json:"method,omitempty"`
		DataType  string     `json:"dataType,omitempty"`
		Value     string     `json:"value,omitempty"`
		Source    Location   `json:"source"`
		Static    *Static    `json:"static,omitempty"`
		Uses      []Location `json:"uses,omitempty"`
		Edges     []Edge     `json:"edges"`
		Pure      bool       `json:"pure"`
		ValueType string     `json:"valueType,omitempty"`
		Effect    string     `json:"effect"`
	}
	nodes := []record{}
	for _, n := range p.Nodes {
		nodes = append(nodes, record{n.ID, n.Type, n.Name, n.Method, n.DataType, n.Value, Location{n.File, n.Line}, n.Static, n.Uses, n.Edges(), n.Pure, n.ValueType, n.Effect})
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(struct {
		Root  int      `json:"root"`
		Nodes []record `json:"nodes"`
	}{p.Root.ID, nodes})
}
