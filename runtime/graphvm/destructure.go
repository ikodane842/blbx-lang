// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"blbx_lang/runtime/graph"
	"blbx_lang/syntax/diagnostic"
)

type patternBinding struct {
	name  string
	value Value
}

// Validate and collect first so a missing field cannot partially assign names.
func collectPattern(pattern *graph.Node, value Value, bindings *[]patternBinding) error {
	switch pattern.Type {
	case graph.Identifier:
		*bindings = append(*bindings, patternBinding{pattern.Name, value})
	case graph.ArrayPattern:
		if value.Kind != ArrayKind && value.Kind != TupleKind {
			return runtimeError(diagnostic.ArgumentType, pattern, "array destructuring requires array, got %s", value.Kind)
		}
		for index, child := range pattern.Inputs {
			if child.Type == graph.RestPattern {
				return collectPattern(child.Inputs[0], Array(append([]Value{}, value.elements()[index:]...)), bindings)
			}
			if index >= len(value.elements()) {
				return runtimeError(diagnostic.DestructureMissing, child, "missing array element %d in destructuring", index)
			}
			if err := collectPattern(child, value.elements()[index], bindings); err != nil {
				return err
			}
		}
	case graph.ObjectPattern:
		if value.Kind != ObjectKind {
			return runtimeError(diagnostic.ArgumentType, pattern, "object destructuring requires object, got %s", value.Kind)
		}
		used := map[string]bool{}
		for _, child := range pattern.Inputs {
			if child.Type == graph.RestPattern {
				rest := map[string]Value{}
				for key, item := range value.Object {
					if !used[key] {
						rest[key] = bindReceiver(item, value)
					}
				}
				return collectPattern(child.Inputs[0], Object(rest), bindings)
			}
			item, exists := value.Object[child.Name]
			if !exists {
				return runtimeError(diagnostic.DestructureMissing, child, "missing object field %q in destructuring", child.Name)
			}
			used[child.Name] = true
			if err := collectPattern(child.Inputs[0], bindReceiver(item, value), bindings); err != nil {
				return err
			}
		}
	}
	return nil
}
