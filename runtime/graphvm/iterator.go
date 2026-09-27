// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"blbx_lang/runtime/graph"
	"blbx_lang/syntax/diagnostic"
)

// A next function returns (value, done, error), so a null element is never
// confused with exhaustion. Iteration is lazy and break does not request again.
func (i *VM) iterator(value Value, node *graph.Node) (func() (Value, bool, error), error) {
	if value.Kind == StringKind {
		values := []Value{}
		for _, r := range value.String {
			values = append(values, String(string(r)))
		}
		value = Array(values)
	}
	if value.Kind == ArrayKind || value.Kind == TupleKind {
		index := 0
		return func() (Value, bool, error) {
			if index >= len(value.elements()) {
				return Null(), true, nil
			}
			item := value.elements()[index]
			index++
			return item, false, nil
		}, nil
	}
	if value.Kind == ObjectKind {
		if method, ok := value.Object["iter"]; ok {
			if method.Kind != FunctionKind {
				return nil, runtimeError(diagnostic.IteratorProtocol, node, "iter must be a function")
			}
			result, err := i.callValue(bindReceiver(method, value), nil)
			if err != nil {
				return nil, err
			}
			value = result.value
		}
		if value.Kind == ObjectKind {
			next, ok := value.Object["next"]
			if ok && next.Kind == FunctionKind {
				next = bindReceiver(next, value)
				return func() (Value, bool, error) {
					result, err := i.callValue(next, nil)
					if err != nil {
						return Null(), false, err
					}
					if result.value.Kind != ObjectKind {
						return Null(), false, runtimeError(diagnostic.IteratorProtocol, node, "next() must return an object with boolean done")
					}
					done, ok := result.value.Object["done"]
					if !ok || done.Kind != BooleanKind {
						return Null(), false, runtimeError(diagnostic.IteratorProtocol, node, "next() must return boolean done")
					}
					if done.Boolean {
						return Null(), true, nil
					}
					item, ok := result.value.Object["value"]
					if !ok {
						return Null(), false, runtimeError(diagnostic.IteratorProtocol, node, "next() must return value when done is false")
					}
					return item, false, nil
				}, nil
			}
		}
	}
	return nil, runtimeError(diagnostic.IteratorProtocol, node, "value is not iterable; expected array, tuple, string, or iter()/next() object")
}
