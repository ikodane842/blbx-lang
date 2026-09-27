package graph

import (
	"fmt"
)

// A binding version is a completed assignment's immutable scalar result, not a
// promise to reread a mutable name later. Versions only flow within a straight
// line region. Unknown calls/control regions invalidate the name environment.
type valueAnalysis struct{ shared map[string]*Node }
type versions map[string]*Node

func (v versions) invalidate() {
	for name := range v {
		delete(v, name)
	}
}

func (p *Program) separateValuesAndEffects() {
	for _, n := range p.Nodes {
		n.Effect = "ordered"
		if n.Type == Literal {
			n.Pure, n.ValueType, n.Effect = true, n.DataType, "none"
		}
		// Even contexts intentionally excluded from value analysis (such as
		// assignment targets) can contain executable callback regions.
		if n.Type == ProgramRegion || n.Type == Block {
			wireRegion(n)
		}
	}
	a := &valueAnalysis{shared: map[string]*Node{}}
	a.expression(p.Root, versions{})
	// Argument demand is guarded by the call's receiver/arity checks. Within
	// that demand, explicit tokens preserve left-to-right nested effects.
	for _, n := range p.Nodes {
		switch n.Type {
		case Call, Array, Tuple, Binary:
			n.ArgumentPlan = p.arguments(n, n.Inputs)
		case MethodCall, ExpressionCall:
			n.ArgumentPlan = p.arguments(n, n.Arguments)
		}
	}
}

func (p *Program) arguments(owner *Node, inputs []*Node) *Node {
	makeNode := func(kind Type, line int) *Node {
		n := &Node{ID: len(p.Nodes) + 1, Type: kind, File: owner.File, Line: line, Effect: "sequence"}
		p.Nodes = append(p.Nodes, n)
		return n
	}
	region := makeNode(ArgumentRegion, owner.Line)
	var previous *Node
	for _, input := range inputs {
		step := makeNode(Step, input.Line)
		step.Inputs = []*Node{input}
		step.EffectIn = previous
		if previous == nil {
			region.Entry = step
		} else {
			previous.Next = step
		}
		previous = step
	}
	region.Exit = previous
	return region
}

func scalar(kind string) bool {
	return kind == "integer" || kind == "float" || kind == "boolean" || kind == "string" || kind == "null"
}

func numeric(kind string) bool { return kind == "integer" || kind == "float" }

func canonical(n *Node) *Node {
	if n.Shared != nil {
		return n.Shared
	}
	return n
}

// share keeps occurrence nodes (and their diagnostics) distinct, but gives pure
// equivalent calculations one value identity. Errors are never cached by the VM.
func (a *valueAnalysis) share(n *Node, inputs ...*Node) {
	key := string(n.Type) + ":" + n.Name + ":" + n.Method + ":" + n.ValueType
	if n.Definition != nil {
		key += fmt.Sprintf(":version=%d", n.Definition.ID)
	}
	for _, input := range inputs {
		if input == nil || !input.Pure {
			return
		}
		key += fmt.Sprintf(":%d", canonical(input).ID)
	}
	n.Pure, n.Effect = true, "none"
	if existing := a.shared[key]; existing != nil {
		n.Shared = existing
	} else {
		a.shared[key] = n
	}
}

func (a *valueAnalysis) region(n *Node) {
	env := versions{}
	for step := n.Entry; step != nil; step = step.Next {
		a.expression(step.Inputs[0], env)
	}
}

func wireRegion(n *Node) {
	var previous *Node
	for step := n.Entry; step != nil; step = step.Next {
		step.Effect = "sequence"
		step.EffectIn = previous
		previous = step
	}
	n.Exit = previous
	n.Effect = "control"
}

func (a *valueAnalysis) expression(n *Node, env versions) {
	if n == nil {
		return
	}
	switch n.Type {
	case Literal:
		return
	case ProgramRegion, Block:
		a.region(n)
		env.invalidate()
	case Identifier:
		n.Effect = "read"
		if definition := env[n.Name]; definition != nil {
			n.Type, n.Definition, n.ValueType = ValueRef, definition, definition.ValueType
			a.share(n)
		}
	case Assign:
		n.Effect = "write"
		if len(n.Inputs) < 2 {
			env.invalidate()
			return
		}
		// BLBX evaluates the RHS before member/index assignment targets.
		rhs := n.Inputs[len(n.Inputs)-1]
		a.expression(rhs, env)
		target := n.Inputs[0]
		if target.Type == Identifier {
			delete(env, target.Name)
			// An effect can produce an immutable scalar too. Its completed
			// result becomes a value input without making the producer pure.
			if scalar(rhs.ValueType) {
				n.ValueType = rhs.ValueType
				env[target.Name] = n
			}
		} else {
			env.invalidate()
		}
	case MethodCall:
		// The receiver is evaluated before arguments, including effectful ones.
		a.expression(n.Receiver, env)
		for _, arg := range n.Arguments {
			a.expression(arg, env)
		}
		if kind := scalarMethodType(n); kind != "" {
			n.ValueType = kind
			a.share(n, append([]*Node{n.Receiver}, n.Arguments...)...)
		}
		if !n.Pure {
			n.Effect = "call"
			env.invalidate()
		}
	case Binary:
		for _, input := range n.Inputs {
			a.expression(input, env)
		}
		if len(n.Inputs) == 2 && n.Name == "*" && numeric(n.Inputs[0].ValueType) && numeric(n.Inputs[1].ValueType) {
			n.ValueType = "integer"
			if n.Inputs[0].ValueType == "float" || n.Inputs[1].ValueType == "float" {
				n.ValueType = "float"
			}
			a.share(n, n.Inputs...)
		}
		if !n.Pure {
			n.Effect = "call"
			env.invalidate()
		}
	case Call:
		for _, input := range n.Inputs {
			a.expression(input, env)
		}
		n.Effect = "call"
		if n.DataType == "direct-call" {
			switch n.Name {
			case "input", "typeof":
				n.ValueType = "string"
			case "ord":
				n.ValueType = "integer"
			case "print":
				n.ValueType = "null"
			}
		}
		if n.DataType == "direct-call" && len(n.Inputs) == 1 && n.Inputs[0].Pure && scalar(n.Inputs[0].ValueType) {
			if n.Name == "typeof" {
				n.ValueType = "string"
				a.share(n, n.Inputs...)
			}
			if n.Name == "ord" && n.Inputs[0].ValueType == "string" {
				n.ValueType = "integer"
				a.share(n, n.Inputs...)
			}
		}
		// These builtins cannot invoke user code themselves. Their argument
		// traversal above already invalidates versions if an argument can.
		if !n.Pure && !(n.DataType == "direct-call" && (n.Name == "print" || n.Name == "input" || n.Name == "typeof" || n.Name == "ord")) {
			env.invalidate()
		}
	case ExpressionCall:
		a.expression(n.Callee, env)
		for _, input := range n.Arguments {
			a.expression(input, env)
		}
		n.Effect = "call"
		env.invalidate()
	case If:
		a.expression(n.Condition, env)
		// Branch-local versions must not escape a conditional merge.
		a.expression(n.Then, versions{})
		a.expression(n.Else, versions{})
		n.Effect = "control"
		env.invalidate()
	case While, For:
		// A loop condition must read changing state on every activation.
		a.expression(n.Condition, versions{})
		a.expression(n.Iterable, versions{})
		a.expression(n.Body, versions{})
		if n.Entry != nil {
			n.Entry.Effect = "control"
		}
		n.Effect = "control"
		env.invalidate()
	case Function:
		// Never capture outer scalar versions into closures: BLBX closures
		// retain environments, and may execute after those names are changed.
		for _, input := range n.Inputs {
			a.expression(input, versions{})
		}
		n.Effect = "allocate"
		env.invalidate()
	case Object, Class, Interface:
		// Field initializers and declaration metadata are not sequence writes.
		// Each gets its own analysis environment, so no phantom binding
		// version can refer to an assignment the VM never executes directly.
		for _, input := range n.Inputs {
			a.expression(input, versions{})
		}
		for _, base := range n.Bases {
			a.expression(base, versions{})
		}
		for _, contract := range n.Interfaces {
			a.expression(contract, versions{})
		}
		a.expression(n.Base, versions{})
		n.Effect = "allocate"
		env.invalidate()
	case Array, Tuple:
		for _, input := range n.Inputs {
			a.expression(input, env)
		}
		n.Effect = "allocate"
		env.invalidate()
	case Member, Index:
		// Reads from mutable state are not cacheable. Member names are
		// reference metadata, not identifier reads.
		for index, input := range n.Inputs {
			if n.Type == Member && index > 0 {
				continue
			}
			a.expression(input, env)
		}
		n.Effect = "read"
	case Return, Assert:
		for _, input := range n.Inputs {
			a.expression(input, env)
		}
		n.Effect = "control"
		env.invalidate()
	default:
		n.Effect = "ordered"
		env.invalidate()
	}
}

// Only immutable, builtin scalar operations are admitted. Unknown receivers,
// overloaded methods, allocation, and native/user functions stay on the effect
// side. Nullable/ambiguous result types are deliberately left uninferred.
func scalarMethodType(n *Node) string {
	if n.Receiver == nil || !scalar(n.Receiver.ValueType) {
		return ""
	}
	for _, arg := range n.Arguments {
		if !scalar(arg.ValueType) {
			return ""
		}
	}
	t := n.Receiver.ValueType
	args := n.Arguments
	if len(args) == 0 {
		switch n.Method {
		case "typeof", "type", "to_str":
			return "string"
		case "to_bool":
			return "boolean"
		case "not":
			if t == "boolean" {
				return "boolean"
			}
		case "bit_not":
			if t == "integer" {
				return "integer"
			}
		case "length":
			if t == "string" {
				return "integer"
			}
		case "is_empty":
			if t == "string" {
				return "boolean"
			}
		case "upper", "lower", "trim", "trim_start", "trim_end", "reverse":
			if t == "string" {
				return "string"
			}
		}
	}
	if t == "string" && n.Method == "concat" {
		return "string"
	}
	if len(args) != 1 {
		return ""
	}
	u := args[0].ValueType
	switch n.Method {
	case "eq", "neq":
		return "boolean"
	case "and", "or":
		if t == "boolean" && u == "boolean" {
			return "boolean"
		}
	case "add", "sub", "mul":
		if numeric(t) && numeric(u) {
			if t == "float" || u == "float" {
				return "float"
			}
			return "integer"
		}
	case "lt", "lte", "gt", "gte":
		if (numeric(t) && numeric(u)) || (t == "string" && u == "string") {
			return "boolean"
		}
	case "bit_and", "bit_or", "bit_xor", "shl", "shr", "ushr":
		if t == "integer" && u == "integer" {
			return "integer"
		}
	case "starts_with", "ends_with", "contains":
		if t == "string" && u == "string" {
			return "boolean"
		}
	case "index_of":
		if t == "string" && u == "string" {
			return "integer"
		}
	}
	return ""
}

// ValueIdentity is useful to graph consumers without requiring them to know
// whether an occurrence was deduplicated by the conservative scalar analysis.
func (n *Node) ValueIdentity() *Node { return canonical(n) }
