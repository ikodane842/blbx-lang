package interpreter

import (
	"blbx_lang/syntax/diagnostic"
	"fmt"
)

type nativeAdder func(string, []Kind, func([]Value) (Value, error))

func contextNative(name, code string, kinds []Kind, fn func(*Interpreter, []Value) (Value, error)) Value {
	return FunctionValue(&Function{Name: name, NativeContext: func(i *Interpreter, args []Value) (Value, error) {
		if len(args) != len(kinds) {
			return Null(), diagnostic.RuntimeCode(diagnostic.ArgumentCount, fmt.Errorf("%s expects %d arguments, got %d", name, len(kinds), len(args)))
		}
		for n, kind := range kinds {
			if kind != "" && args[n].Kind != kind {
				return Null(), diagnostic.RuntimeCode(diagnostic.ArgumentType, fmt.Errorf("%s argument %d must be %s", name, n+1, kind))
			}
		}
		value, err := fn(i, args)
		if err != nil {
			return Null(), diagnostic.RuntimeCode(code, fmt.Errorf("%s: %w", name, err))
		}
		return value, nil
	}})
}

func byteArray(data []byte) Value {
	values := make([]Value, len(data))
	for n, b := range data {
		values[n] = Integer(int64(b))
	}
	return Array(values)
}
func valueBytes(value Value) ([]byte, error) {
	if value.Kind == StringKind {
		if len(value.String) > stdOutputLimit {
			return nil, fmt.Errorf("data exceeds 8 MiB")
		}
		return []byte(value.String), nil
	}
	if value.Kind != ArrayKind {
		return nil, fmt.Errorf("expected a string or byte array")
	}
	if len(value.Array) > stdOutputLimit {
		return nil, fmt.Errorf("data exceeds 8 MiB")
	}
	data := make([]byte, len(value.Array))
	for n, v := range value.Array {
		if v.Kind != IntegerKind || v.Integer < 0 || v.Integer > 255 {
			return nil, fmt.Errorf("byte %d must be an integer from 0 to 255", n)
		}
		data[n] = byte(v.Integer)
	}
	return data, nil
}
