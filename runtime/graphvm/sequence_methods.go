// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Sequence operations return new values. Arrays are shallow copies: nested
// objects retain their identity, but the source array's elements are unchanged.
func sequenceMethod(receiver Value, method string, args []Value) (Value, bool) {
	if receiver.Kind == TupleKind {
		receiver.Kind = ArrayKind
		value, ok := sequenceMethod(receiver, method, args)
		if value.Kind == ArrayKind {
			value.Kind = TupleKind
		}
		return value, ok
	}
	if receiver.Kind != StringKind && receiver.Kind != ArrayKind {
		return Null(), false
	}
	switch method {
	case "is_empty":
		if len(args) == 0 {
			if receiver.Kind == StringKind {
				return Boolean(receiver.String == ""), true
			}
			return Boolean(len(receiver.elements()) == 0), true
		}
	case "slice":
		if receiver.Kind == StringKind {
			runes := []rune(receiver.String)
			start, end, ok := sliceBounds(len(runes), args)
			if ok {
				return String(string(runes[start:end])), true
			}
		} else {
			start, end, ok := sliceBounds(len(receiver.elements()), args)
			if ok {
				return Array(append([]Value{}, receiver.elements()[start:end]...)), true
			}
		}
	case "reverse":
		if len(args) == 0 {
			if receiver.Kind == StringKind {
				runes := []rune(receiver.String)
				for left, right := 0, len(runes)-1; left < right; left, right = left+1, right-1 {
					runes[left], runes[right] = runes[right], runes[left]
				}
				return String(string(runes)), true
			}
			values := append([]Value{}, receiver.elements()...)
			for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
				values[left], values[right] = values[right], values[left]
			}
			return Array(values), true
		}
	case "contains", "index_of":
		if len(args) == 1 {
			index := -1
			if receiver.Kind == StringKind {
				if args[0].Kind != StringKind {
					return Null(), true
				}
				byteIndex := strings.Index(receiver.String, args[0].String)
				if byteIndex >= 0 {
					index = utf8.RuneCountInString(receiver.String[:byteIndex])
				}
			} else {
				for n, value := range receiver.elements() {
					if value.Equal(args[0]) {
						index = n
						break
					}
				}
			}
			if method == "contains" {
				return Boolean(index >= 0), true
			}
			return Integer(int64(index)), true
		}
	case "concat":
		if receiver.Kind == StringKind {
			var result strings.Builder
			result.WriteString(receiver.String)
			for _, value := range args {
				result.WriteString(value.Display())
			}
			return String(result.String()), true
		}
		values := append([]Value{}, receiver.elements()...)
		for _, value := range args {
			if value.Kind != ArrayKind {
				return Null(), true
			}
			values = append(values, value.elements()...)
		}
		return Array(values), true
	case "join":
		if receiver.Kind == ArrayKind {
			separator := ""
			if len(args) > 1 {
				return Null(), true
			}
			if len(args) == 1 {
				if args[0].Kind != StringKind {
					return Null(), true
				}
				separator = args[0].String
			}
			return joinValues(receiver.elements(), separator), true
		}
		if len(args) == 1 && args[0].Kind == ArrayKind {
			return joinValues(args[0].elements(), receiver.String), true
		}
	default:
		if receiver.Kind == StringKind {
			return stringMethod(receiver.String, method, args)
		}
		return arrayMethod(receiver.elements(), method, args)
	}
	return Null(), true
}

func joinValues(values []Value, separator string) Value {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = value.Display()
	}
	return String(strings.Join(parts, separator))
}

func stringMethod(value, method string, args []Value) (Value, bool) {
	switch method {
	case "split":
		var parts []string
		if len(args) == 0 {
			parts = strings.Fields(value)
		} else if len(args) == 1 && args[0].Kind == StringKind {
			parts = strings.Split(value, args[0].String)
		} else {
			return Null(), true
		}
		values := make([]Value, len(parts))
		for index, part := range parts {
			values[index] = String(part)
		}
		return Array(values), true
	case "upper", "lower":
		if len(args) == 0 {
			if method == "upper" {
				return String(strings.ToUpper(value)), true
			}
			return String(strings.ToLower(value)), true
		}
	case "trim", "trim_start", "trim_end":
		if len(args) == 0 {
			switch method {
			case "trim":
				return String(strings.TrimSpace(value)), true
			case "trim_start":
				return String(strings.TrimLeftFunc(value, unicode.IsSpace)), true
			case "trim_end":
				return String(strings.TrimRightFunc(value, unicode.IsSpace)), true
			}
		}
	case "starts_with", "ends_with":
		if len(args) == 1 && args[0].Kind == StringKind {
			if method == "starts_with" {
				return Boolean(strings.HasPrefix(value, args[0].String)), true
			}
			return Boolean(strings.HasSuffix(value, args[0].String)), true
		}
	case "replace", "replace_all":
		if len(args) == 2 && args[0].Kind == StringKind && args[1].Kind == StringKind {
			count := 1
			if method == "replace_all" {
				count = -1
			}
			return String(strings.Replace(value, args[0].String, args[1].String, count)), true
		}
	case "char_at":
		if len(args) == 1 && args[0].Kind == IntegerKind {
			runes := []rune(value)
			index := args[0].Integer
			if index < 0 {
				index += int64(len(runes))
			}
			if index >= 0 && index < int64(len(runes)) {
				return String(string(runes[index])), true
			}
		}
	case "repeat":
		if len(args) == 1 && args[0].Kind == IntegerKind && args[0].Integer >= 0 {
			if value == "" {
				return String(""), true
			}
			// Bound expansion and avoid strings.Repeat panicking on overflow.
			if args[0].Integer <= int64((16<<20)/len(value)) {
				return String(strings.Repeat(value, int(args[0].Integer))), true
			}
		}
	default:
		return Null(), false
	}
	return Null(), true
}

func arrayMethod(values []Value, method string, args []Value) (Value, bool) {
	switch method {
	case "append":
		return Array(append(append([]Value{}, values...), args...)), true
	case "first", "last":
		if len(args) == 0 && len(values) > 0 {
			if method == "first" {
				return values[0], true
			}
			return values[len(values)-1], true
		}
	default:
		return Null(), false
	}
	return Null(), true
}

func sliceBounds(length int, args []Value) (int, int, bool) {
	if len(args) > 2 {
		return 0, 0, false
	}
	indices := []int{0, length}
	for index, value := range args {
		if value.Kind != IntegerKind {
			return 0, 0, false
		}
		n := value.Integer
		if n < 0 {
			n += int64(length)
		}
		if n < 0 {
			n = 0
		}
		if n > int64(length) {
			n = int64(length)
		}
		indices[index] = int(n)
	}
	if indices[1] < indices[0] {
		indices[1] = indices[0]
	}
	return indices[0], indices[1], true
}
