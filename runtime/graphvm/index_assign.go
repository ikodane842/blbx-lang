// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"blbx_lang/runtime/graph"
	"blbx_lang/syntax/diagnostic"
)

func (i *VM) assignIndex(node *graph.Node, value Value, scope *Scope) error {
	target, err := i.eval(node.Inputs[0], scope)
	if err != nil {
		return err
	}
	index, err := i.eval(node.Inputs[1], scope)
	if err != nil {
		return err
	}
	switch target.value.Kind {
	case ObjectKind:
		if index.value.Kind != StringKind {
			return runtimeError(diagnostic.ArgumentType, node, "object index must be a string")
		}
		target.value.Object[index.value.String] = value
		return nil
	case ArrayKind:
		if index.value.Kind != IntegerKind {
			return runtimeError(diagnostic.ArgumentType, node, "array index must be an integer")
		}
		n := index.value.Integer
		if n < 0 || n >= int64(len(target.value.elements())) {
			return runtimeError(diagnostic.IndexRange, node, "array assignment index %d is out of range", n)
		}
		target.value.elements()[n] = value
		return nil
	default:
		return runtimeError(diagnostic.InvalidReceiver, node, "indexed assignment is not available on %s", target.value.Kind)
	}
}
