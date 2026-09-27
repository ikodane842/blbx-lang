// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"blbx_lang/runtime/graph"
	"blbx_lang/syntax/diagnostic"
)

// Signature validation runs before this helper; operands are evaluated once.
func bitwiseMethod(left int64, method string, args []Value, node *graph.Node) (Value, bool, error) {
	switch method {
	case "bit_not":
		return Integer(^left), true, nil
	case "bit_and", "bit_or", "bit_xor", "shl", "shr", "ushr":
	default:
		return Null(), false, nil
	}
	if args[0].Kind != IntegerKind {
		return Null(), true, runtimeError(diagnostic.ArgumentType, node, "%s requires an integer argument", method)
	}
	right := args[0].Integer
	switch method {
	case "bit_and":
		return Integer(left & right), true, nil
	case "bit_or":
		return Integer(left | right), true, nil
	case "bit_xor":
		return Integer(left ^ right), true, nil
	}
	if right < 0 || right > 63 {
		return Null(), true, runtimeError(diagnostic.ShiftRange, node, "%s shift count must be between 0 and 63, got %d", method, right)
	}
	switch method {
	case "shl":
		return Integer(left << uint(right)), true, nil
	case "shr":
		return Integer(left >> uint(right)), true, nil
	default: // ushr shifts the bit pattern with zero fill, then returns an integer.
		return Integer(int64(uint64(left) >> uint(right))), true, nil
	}
}
