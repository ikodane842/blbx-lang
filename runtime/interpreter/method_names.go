package interpreter

// Recognition is separate from evaluating a method: a valid method may itself
// return null, for example a failed conversion or first() on an empty array.
func knownMethod(name string) bool {
	switch name {
	case "to_str", "to_int", "to_float", "concat", "add", "sub", "mul", "div", "mod",
		"gt", "gte", "lt", "lte", "eq", "neq", "and", "or", "not", "type", "typeof", "length",
		"elem", "idx", "index", "step", "continue", "break", "indexes", "values",
		"is_empty", "slice", "reverse", "contains", "index_of", "join", "split",
		"upper", "lower", "trim", "trim_start", "trim_end", "starts_with", "ends_with",
		"replace", "replace_all", "char_at", "repeat", "append", "first", "last":
		return true
	}
	return false
}
