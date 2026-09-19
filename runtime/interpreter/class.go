package interpreter

import (
	"blbx_lang/syntax/diagnostic"
	"blbx_lang/syntax/ir"
)

// Bind a copy so calling a shared method on one object never changes another
// object's receiver. Extracted methods retain their receiver as well.
func bindReceiver(value, receiver Value) Value {
	if value.Kind != FunctionKind || value.Function == nil {
		return value
	}
	fn := *value.Function
	fn.Env = NewScope(fn.Env)
	fn.Env.Define("self", receiver)
	if fn.Owner != nil {
		fn.Env.Define("super", Value{Kind: ObjectKind, Super: &superReference{Receiver: receiver, Owner: fn.Owner}})
	}
	return FunctionValue(&fn)
}

func (i *Interpreter) callValue(value Value, args []Value) (evalResult, error) {
	if value.Kind == ClassKind {
		return i.instantiate(value.Class, args)
	}
	if value.Kind != FunctionKind || value.Function == nil {
		return evalResult{}, runtimeError(diagnostic.NotCallable, ir.Node{}, "%s value is not callable", value.Kind)
	}
	return i.callFunction(value.Function, args)
}

func (i *Interpreter) instantiate(class *Class, args []Value) (evalResult, error) {
	instance := Object(map[string]Value{})
	instance.Class = class
	instance.Layers = map[*Class]map[string]Value{}
	var constructor *ir.Node
	var constructorScope *Scope
	var constructorOwner *Class
	lineage := class.MRO
	// Evaluate base defaults first. Each declaring class keeps its lexical
	// environment, while every method operates on the same derived instance.
	for level := len(lineage) - 1; level >= 0; level-- {
		declaring := lineage[level]
		instance.Layers[declaring] = map[string]Value{}
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
					return evalResult{}, runtimeError(diagnostic.DuplicateConstructor, *member, "class may contain only one constructor")
				}
				ownConstructor = member
				continue
			}
			if member.Type != ir.Assign || len(member.Children) < 2 || member.Children[0].Type != ir.Identifier {
				return evalResult{}, runtimeError(diagnostic.InvalidClass, *member, "invalid class member")
			}
			value, err := i.evalAssignmentValue(member.Children[len(member.Children)-1], fields)
			if err != nil || value.signal != noSignal {
				return value, err
			}
			if value.value.Kind == FunctionKind {
				copy := *value.value.Function
				copy.Owner = declaring
				value.value.Function = &copy
			}
			fields.Define(member.Children[0].Name, value.value)
			instance.Layers[declaring][member.Children[0].Name] = value.value
		}
		if ownConstructor != nil {
			constructor, constructorScope = ownConstructor, fields
			constructorOwner = declaring
		}
	}
	if constructor != nil {
		fn, err := i.evalFunction(*constructor, constructorScope)
		if err != nil {
			return fn, err
		}
		fn.value.Function.Owner = constructorOwner
		result, err := i.callValue(bindReceiver(fn.value, instance), args)
		if err != nil {
			return result, err
		}
		// A constructor's return value never replaces the new instance.
	} else if len(args) != 0 {
		return evalResult{}, runtimeError(diagnostic.ArgumentCount, ir.Node{Name: class.Name}, "class %s has no constructor and accepts no arguments", class.Name)
	}
	for _, contract := range class.Interfaces {
		for name, arity := range contract.Methods {
			method, ok := instance.Object[name]
			if !ok || method.Kind != FunctionKind || len(method.Function.Params) != arity {
				return evalResult{}, runtimeError(diagnostic.InterfaceMismatch, ir.Node{}, "class %s must implement %s.%s with %d parameters", class.Name, contract.Name, name, arity)
			}
		}
	}
	return normal(instance)
}
