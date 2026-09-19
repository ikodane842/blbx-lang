package interpreter

import (
	"blbx_lang/syntax/diagnostic"
	"blbx_lang/syntax/ir"
)

type patternBinding struct {
	name  string
	value Value
}

// Validate and collect first so a missing field cannot partially assign names.
func collectPattern(pattern ir.Node, value Value, bindings *[]patternBinding) error {
	switch pattern.Type {
	case ir.Identifier:
		*bindings = append(*bindings, patternBinding{pattern.Name, value})
	case ir.ArrayPattern:
		if value.Kind != ArrayKind && value.Kind != TupleKind {
			return runtimeError(diagnostic.ArgumentType, pattern, "array destructuring requires array, got %s", value.Kind)
		}
		for index, child := range pattern.Children {
			if child.Type == ir.RestPattern {
				return collectPattern(child.Children[0], Array(append([]Value{}, value.Array[index:]...)), bindings)
			}
			if index >= len(value.Array) {
				return runtimeError(diagnostic.DestructureMissing, child, "missing array element %d in destructuring", index)
			}
			if err := collectPattern(child, value.Array[index], bindings); err != nil {
				return err
			}
		}
	case ir.ObjectPattern:
		if value.Kind != ObjectKind {
			return runtimeError(diagnostic.ArgumentType, pattern, "object destructuring requires object, got %s", value.Kind)
		}
		used := map[string]bool{}
		for _, child := range pattern.Children {
			if child.Type == ir.RestPattern {
				rest := map[string]Value{}
				for key, item := range value.Object {
					if !used[key] {
						rest[key] = bindReceiver(item, value)
					}
				}
				return collectPattern(child.Children[0], Object(rest), bindings)
			}
			item, exists := value.Object[child.Name]
			if !exists {
				return runtimeError(diagnostic.DestructureMissing, child, "missing object field %q in destructuring", child.Name)
			}
			used[child.Name] = true
			if err := collectPattern(child.Children[0], bindReceiver(item, value), bindings); err != nil {
				return err
			}
		}
	}
	return nil
}
