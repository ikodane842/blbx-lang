// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

// Built-in singleton methods are selected by receiver type. A negative maximum
// means variadic. User-defined object methods are resolved separately.
func methodSignature(value Value, name string) (minimum, maximum int, available bool) {
	if value.Kind == TupleKind {
		switch name {
		case "length", "is_empty", "first", "last", "reverse":
			return 0, 0, true
		case "slice":
			return 0, 2, true
		case "contains", "index_of":
			return 1, 1, true
		case "join":
			return 0, 1, true
		}
	}
	switch name {
	case "to_str", "type", "typeof", "to_bool":
		return 0, 0, true
	case "eq", "neq":
		return 1, 1, true
	case "in":
		return 1, 1, value.Kind == StringKind
	case "to_int", "to_float":
		return 0, 0, isNumeric(value) || value.Kind == StringKind
	case "not":
		return 0, 0, value.Kind == BooleanKind
	case "and", "or":
		return 1, 1, value.Kind == BooleanKind
	case "add", "sub", "mul", "div":
		return 1, 1, isNumeric(value)
	case "mod":
		return 1, 1, value.Kind == IntegerKind
	case "bit_and", "bit_or", "bit_xor", "shl", "shr", "ushr":
		return 1, 1, value.Kind == IntegerKind
	case "bit_not":
		return 0, 0, value.Kind == IntegerKind
	case "gt", "gte", "lt", "lte":
		return 1, 1, isNumeric(value) || value.Kind == StringKind
	case "indexes", "values":
		return 0, 0, value.Kind == ObjectKind && value.Class == nil
	case "elem", "idx", "index", "step", "continue", "break":
		return 0, 0, value.Cursor
	}
	if value.Kind == StringKind || value.Kind == ArrayKind {
		switch name {
		case "length", "is_empty", "reverse":
			return 0, 0, true
		case "slice":
			return 0, 2, true
		case "contains", "index_of":
			return 1, 1, true
		case "concat":
			return 0, -1, true
		}
	}
	if value.Kind == StringKind {
		switch name {
		case "split":
			return 0, 1, true
		case "join", "starts_with", "ends_with", "char_at", "repeat":
			return 1, 1, true
		case "upper", "lower", "trim", "trim_start", "trim_end":
			return 0, 0, true
		case "replace", "replace_all":
			return 2, 2, true
		}
	}
	if value.Kind == ArrayKind {
		switch name {
		case "extend":
			return 1, 1, true
		case "join":
			return 0, 1, true
		case "append":
			return 0, -1, true
		case "first", "last":
			return 0, 0, true
		}
	}
	return 0, 0, false
}
