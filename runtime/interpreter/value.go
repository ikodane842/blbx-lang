package interpreter

import (
	"blbx_lang/syntax/ir"
	"encoding/json"
	"strconv"
	"strings"
)

type Kind string

const (
	NullKind      Kind = "null"
	BooleanKind   Kind = "boolean"
	IntegerKind   Kind = "integer"
	FloatKind     Kind = "float"
	StringKind    Kind = "string"
	ArrayKind     Kind = "array"
	TupleKind     Kind = "tuple"
	ObjectKind    Kind = "object"
	FunctionKind  Kind = "function"
	ClassKind     Kind = "class"
	TaskKind      Kind = "task"
	InterfaceKind Kind = "interface"
	ResourceKind  Kind = "resource"
)

type Value struct {
	Resource  *networkResource
	Super     *superReference
	Layers    map[*Class]map[string]Value
	Interface *Interface
	Task      *Task
	Cursor    bool
	Kind      Kind
	Boolean   bool
	Integer   int64
	Float     float64
	String    string
	array     *[]Value
	Object    map[string]Value
	Function  *Function
	Class     *Class
}

type Class struct {
	Parents    []*Class
	MRO        []*Class
	Interfaces []*Interface
	Parent     *Class
	Name       string
	Body       []ir.Node
	Env        *Scope
}

type Function struct {
	Owner         *Class
	NativeContext func(*Interpreter, []Value) (Value, error)
	Native        func([]Value) (Value, error)
	Name          string
	Params        []Param
	Body          ir.Node
	Env           *Scope
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
	return Value{Kind: ArrayKind, array: &values}
}

func (v Value) elements() []Value {
	if v.array == nil {
		return nil
	}
	return *v.array
}

func Object(values map[string]Value) Value {
	return Value{Kind: ObjectKind, Object: values}
}

func FunctionValue(fn *Function) Value {
	return Value{Kind: FunctionKind, Function: fn}
}

func (v Value) IsTruthy() bool {
	if v.Kind == TaskKind {
		return true
	}
	switch v.Kind {
	case BooleanKind:
		return v.Boolean
	case IntegerKind:
		return v.Integer != 0
	case FloatKind:
		return v.Float != 0
	case StringKind:
		return v.String != ""
	case ArrayKind, TupleKind:
		return len(v.elements()) > 0
	case ObjectKind, FunctionKind, ClassKind, InterfaceKind, ResourceKind:
		return true
	default:
		return false
	}
}

func (v Value) TypeName() string {
	return string(v.Kind)
}

func (v Value) Display() string {
	if v.Kind == ResourceKind {
		return "<" + v.Resource.kind + " resource>"
	}
	if v.Kind == InterfaceKind {
		return "<interface " + v.Interface.Name + ">"
	}
	if v.Kind == TaskKind {
		return "<task>"
	}
	switch v.Kind {
	case BooleanKind:
		return strconv.FormatBool(v.Boolean)
	case IntegerKind:
		return strconv.FormatInt(v.Integer, 10)
	case FloatKind:
		return strconv.FormatFloat(v.Float, 'f', -1, 64)
	case StringKind:
		return v.String
	case ArrayKind, TupleKind:
		parts := []string{}
		for _, item := range v.elements() {
			parts = append(parts, item.Display())
		}
		if v.Kind == TupleKind {
			suffix := ""
			if len(parts) == 1 {
				suffix = ","
			}
			return "(" + strings.Join(parts, ", ") + suffix + ")"
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case ObjectKind:
		if v.Class != nil {
			return "<" + v.Class.Name + " instance>"
		}
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
	case ClassKind:
		return "<class " + v.Class.Name + ">"
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
	case TupleKind:
		if len(v.elements()) != len(other.elements()) {
			return false
		}
		for n, item := range v.elements() {
			if !item.Equal(other.elements()[n]) {
				return false
			}
		}
		return true
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
	case ResourceKind:
		return json.Marshal(struct {
			Kind Kind `json:"kind"`
		}{ResourceKind})
	case InterfaceKind:
		return json.Marshal(struct {
			Kind Kind   `json:"kind"`
			Name string `json:"name"`
		}{InterfaceKind, v.Interface.Name})
	case TaskKind:
		return json.Marshal(struct {
			Kind Kind `json:"kind"`
		}{TaskKind})
	case ClassKind:
		return json.Marshal(struct {
			Kind Kind   `json:"kind"`
			Name string `json:"name"`
		}{ClassKind, v.Class.Name})
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
	case ArrayKind, TupleKind:
		return json.Marshal(struct {
			Kind  Kind    `json:"kind"`
			Value []Value `json:"value"`
		}{v.Kind, v.elements()})
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
