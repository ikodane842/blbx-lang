package interpreter

import (
	"blbx_lang/syntax/ir"
	"encoding/json"
	"strconv"
	"strings"
)

type Kind string

const (
	NullKind     Kind = "null"
	BooleanKind  Kind = "boolean"
	IntegerKind  Kind = "integer"
	FloatKind    Kind = "float"
	StringKind   Kind = "string"
	ArrayKind    Kind = "array"
	ObjectKind   Kind = "object"
	FunctionKind Kind = "function"
)

type Value struct {
	Kind     Kind
	Boolean  bool
	Integer  int64
	Float    float64
	String   string
	Array    []Value
	Object   map[string]Value
	Function *Function
}

type Function struct {
	Name   string
	Params []Param
	Body   ir.Node
	Env    *Scope
}

type Param struct {
	Name    string
	Default Value
}

func Null() Value {
	return Value{Kind: NullKind}
}

func Boolean(v bool) Value {
	return Value{Kind: BooleanKind, Boolean: v}
}

func Integer(v int64) Value {
	return Value{Kind: IntegerKind, Integer: v}
}

func Float(v float64) Value {
	return Value{Kind: FloatKind, Float: v}
}

func String(v string) Value {
	return Value{Kind: StringKind, String: v}
}

func Array(values []Value) Value {
	return Value{Kind: ArrayKind, Array: values}
}

func Object(values map[string]Value) Value {
	return Value{Kind: ObjectKind, Object: values}
}

func FunctionValue(fn *Function) Value {
	return Value{Kind: FunctionKind, Function: fn}
}

func (v Value) IsTruthy() bool {
	switch v.Kind {
	case BooleanKind:
		return v.Boolean
	case IntegerKind:
		return v.Integer != 0
	case FloatKind:
		return v.Float != 0
	case StringKind:
		return v.String != ""
	case ArrayKind:
		return len(v.Array) > 0
	case ObjectKind, FunctionKind:
		return true
	default:
		return false
	}
}

func (v Value) TypeName() string {
	return string(v.Kind)
}

func (v Value) Display() string {
	switch v.Kind {
	case BooleanKind:
		return strconv.FormatBool(v.Boolean)
	case IntegerKind:
		return strconv.FormatInt(v.Integer, 10)
	case FloatKind:
		return strconv.FormatFloat(v.Float, 'f', -1, 64)
	case StringKind:
		return v.String
	case ArrayKind:
		parts := []string{}
		for _, item := range v.Array {
			parts = append(parts, item.Display())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case ObjectKind:
		parts := []string{}
		for key, item := range v.Object {
			parts = append(parts, key+": "+item.Display())
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case FunctionKind:
		if v.Function != nil && v.Function.Name != "" {
			return "<function " + v.Function.Name + ">"
		}
		return "<function>"
	default:
		return "null"
	}
}

func (v Value) Equal(other Value) bool {
	if v.Kind != other.Kind {
		if isNumeric(v) && isNumeric(other) {
			return v.number() == other.number()
		}
		return false
	}

	switch v.Kind {
	case BooleanKind:
		return v.Boolean == other.Boolean
	case IntegerKind:
		return v.Integer == other.Integer
	case FloatKind:
		return v.Float == other.Float
	case StringKind:
		return v.String == other.String
	case NullKind:
		return true
	default:
		return false
	}
}

func (v Value) number() float64 {
	switch v.Kind {
	case IntegerKind:
		return float64(v.Integer)
	case FloatKind:
		return v.Float
	default:
		return 0
	}
}

func isNumeric(v Value) bool {
	return v.Kind == IntegerKind || v.Kind == FloatKind
}

func (v Value) MarshalJSON() ([]byte, error) {
	switch v.Kind {
	case BooleanKind:
		return json.Marshal(struct {
			Kind  Kind `json:"kind"`
			Value bool `json:"value"`
		}{v.Kind, v.Boolean})
	case IntegerKind:
		return json.Marshal(struct {
			Kind  Kind  `json:"kind"`
			Value int64 `json:"value"`
		}{v.Kind, v.Integer})
	case FloatKind:
		return json.Marshal(struct {
			Kind  Kind    `json:"kind"`
			Value float64 `json:"value"`
		}{v.Kind, v.Float})
	case StringKind:
		return json.Marshal(struct {
			Kind  Kind   `json:"kind"`
			Value string `json:"value"`
		}{v.Kind, v.String})
	case ArrayKind:
		return json.Marshal(struct {
			Kind  Kind    `json:"kind"`
			Value []Value `json:"value"`
		}{v.Kind, v.Array})
	case ObjectKind:
		return json.Marshal(struct {
			Kind  Kind             `json:"kind"`
			Value map[string]Value `json:"value"`
		}{v.Kind, v.Object})
	case FunctionKind:
		name := ""
		if v.Function != nil {
			name = v.Function.Name
		}
		return json.Marshal(struct {
			Kind Kind   `json:"kind"`
			Name string `json:"name,omitempty"`
		}{v.Kind, name})
	default:
		return json.Marshal(struct {
			Kind Kind `json:"kind"`
		}{NullKind})
	}
}
