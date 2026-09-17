package interpreter

import "blbx_lang/syntax/ir"

// Bind a copy so calling a shared method on one object never changes another
// object's receiver. Extracted methods retain their receiver as well.
func bindReceiver(value, receiver Value) Value {
	if value.Kind != FunctionKind || value.Function == nil {
		return value
	}
	fn := *value.Function
	fn.Env = NewScope(fn.Env)
	fn.Env.Define("self", receiver)
	return FunctionValue(&fn)
}

func (i *Interpreter) callValue(value Value, args []Value) (evalResult, error) {
	if value.Kind == ClassKind {
		return i.instantiate(value.Class, args)
	}
	return i.callFunction(value.Function, args)
}

func (i *Interpreter) instantiate(class *Class, args []Value) (evalResult, error) {
	instance := Object(map[string]Value{})
	instance.Class = class
	var constructor *ir.Node
	var constructorScope *Scope
	lineage := []*Class{}
	for current := class; current != nil; current = current.Parent {
		lineage = append(lineage, current)
	}
	// Evaluate base defaults first. Each declaring class keeps its lexical
	// environment, while every method operates on the same derived instance.
	for level := len(lineage) - 1; level >= 0; level-- {
		declaring := lineage[level]
		selfScope := NewScope(declaring.Env)
		selfScope.Define("self", instance)
		fields := NewScope(selfScope)
		fields.values = instance.Object
		var ownConstructor *ir.Node
		// Initialize every field and method before invoking the constructor, even
		// when the anonymous constructor appears before their declarations.
		for index := range declaring.Body {
			member := &declaring.Body[index]
			if member.Type == ir.Function {
				if ownConstructor != nil {
					return evalResult{}, runtimeError(*member, "class may contain only one constructor")
				}
				ownConstructor = member
				continue
			}
			if member.Type != ir.Assign || len(member.Children) < 2 || member.Children[0].Type != ir.Identifier {
				return evalResult{}, runtimeError(*member, "invalid class member")
			}
			value, err := i.evalAssignmentValue(member.Children[len(member.Children)-1], fields)
			if err != nil || value.signal != noSignal {
				return value, err
			}
			fields.Define(member.Children[0].Name, value.value)
		}
		if ownConstructor != nil {
			constructor, constructorScope = ownConstructor, fields
		}
	}
	if constructor != nil {
		fn, err := i.evalFunction(*constructor, constructorScope)
		if err != nil {
			return fn, err
		}
		result, err := i.callFunction(fn.value.Function, args)
		if err != nil {
			return result, err
		}
		// A constructor's return value never replaces the new instance.
	} else if len(args) != 0 {
		return evalResult{}, runtimeError(ir.Node{Name: class.Name}, "class %s has no constructor and accepts no arguments", class.Name)
	}
	return normal(instance)
}
