// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"blbx_lang/runtime/graph"
	"blbx_lang/syntax/diagnostic"
)

type Interface struct {
	Name    string
	Methods map[string]int
}
type superReference struct {
	Receiver Value
	Owner    *Class
}

func (i *VM) evalClass(node *graph.Node, scope *Scope) (evalResult, error) {
	class := &Class{Name: node.Name, Body: node.Inputs, Env: scope}
	bases := node.Bases
	if node.Base != nil {
		bases = append([]*graph.Node{node.Base}, bases...)
	}
	seen := map[*Class]bool{}
	for _, b := range bases {
		result, err := i.eval(b, scope)
		if err != nil {
			return result, err
		}
		parent := result.value.Class
		if result.value.Kind != ClassKind || parent == nil {
			return evalResult{}, runtimeError(diagnostic.InvalidBase, node, "base of class %s must be a class", node.Name)
		}
		if seen[parent] {
			return evalResult{}, runtimeError(diagnostic.InheritanceOrder, node, "duplicate base class %s", parent.Name)
		}
		seen[parent] = true
		class.Parents = append(class.Parents, parent)
		class.Interfaces = append(class.Interfaces, parent.Interfaces...)
	}
	if len(class.Parents) > 0 {
		class.Parent = class.Parents[0]
	}
	sequences := [][]*Class{}
	for _, p := range class.Parents {
		sequences = append(sequences, append([]*Class{}, p.MRO...))
	}
	sequences = append(sequences, append([]*Class{}, class.Parents...))
	class.MRO = []*Class{class}
	for {
		remaining := false
		var candidate *Class
		for _, seq := range sequences {
			if len(seq) == 0 {
				continue
			}
			remaining = true
			head := seq[0]
			valid := true
			for _, other := range sequences {
				for n := 1; n < len(other); n++ {
					if other[n] == head {
						valid = false
					}
				}
			}
			if valid {
				candidate = head
				break
			}
		}
		if !remaining {
			break
		}
		if candidate == nil {
			return evalResult{}, runtimeError(diagnostic.InheritanceOrder, node, "inconsistent method order for class %s", node.Name)
		}
		class.MRO = append(class.MRO, candidate)
		for n, seq := range sequences {
			if len(seq) > 0 && seq[0] == candidate {
				sequences[n] = seq[1:]
			}
		}
	}
	for _, n := range node.Interfaces {
		result, err := i.eval(n, scope)
		if err != nil {
			return result, err
		}
		if result.value.Kind != InterfaceKind {
			return evalResult{}, runtimeError(diagnostic.InterfaceMismatch, node, "implements requires an interface")
		}
		class.Interfaces = append(class.Interfaces, result.value.Interface)
	}
	value := Value{Kind: ClassKind, Class: class}
	scope.Define(node.Name, value)
	return normal(value)
}

func (i *VM) superMember(proxy *superReference, name string, node *graph.Node) (Value, error) {
	if proxy.Receiver.Class == nil {
		return Null(), runtimeError(diagnostic.InvalidSelf, node, "super requires a class instance receiver")
	}
	active := false
	for _, class := range proxy.Receiver.Class.MRO {
		if !active {
			if class == proxy.Owner {
				active = true
			}
			continue
		}
		if name != "" {
			if value, ok := proxy.Receiver.Layers[class][name]; ok {
				return bindReceiver(value, proxy.Receiver), nil
			}
		} else {
			for _, member := range class.Body {
				if member.Type != graph.Function {
					continue
				}
				scope := NewScope(class.Env)
				scope.Define("self", proxy.Receiver)
				fields := NewScope(scope)
				fields.values = proxy.Receiver.Object
				result, err := i.evalFunction(member, fields)
				if err != nil {
					return Null(), err
				}
				result.value.Function.Owner = class
				return bindReceiver(result.value, proxy.Receiver), nil
			}
		}
	}
	if name == "" {
		return FunctionValue(&Function{Native: func(args []Value) (Value, error) {
			if len(args) > 0 {
				return Null(), runtimeError(diagnostic.ArgumentCount, node, "super constructor accepts no arguments")
			}
			return Null(), nil
		}}), nil
	}
	return Null(), runtimeError(diagnostic.MissingMember, node, "super has no member %q", name)
}
