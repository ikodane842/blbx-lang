package interpreter

import (
	"blbx_lang/syntax/diagnostic"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Native modules use the same function values and imports as source modules.
// The std namespace is reserved; no I/O happens merely by importing a module.
func standardModule(moduleName string) (Value, bool) {
	return standardModuleAt(moduleName, "")
}

func standardModuleAt(moduleName, baseDir string) (Value, bool) {
	if moduleName == "std.task" {
		return taskModule(), true
	}
	resolve := func(path string) string {
		if baseDir == "" || filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(baseDir, path)
	}
	exports := map[string]Value{}
	failureCode := map[string]string{
		"std.strings": diagnostic.StringsFailure, "std.files": diagnostic.FilesFailure,
		"std.time": diagnostic.TimeFailure, "std.networking": diagnostic.NetworkFailure,
		"std.collections": diagnostic.CollectionsFailure, "std.serialization": diagnostic.SerializationFailure,
		"std.math": diagnostic.MathFailure, "std.processes": diagnostic.ProcessFailure,
		"std.os": diagnostic.OSFailure, "std.security": diagnostic.SecurityFailure,
	}[moduleName]
	add := func(name string, kinds []Kind, fn func([]Value) (Value, error)) {
		qualified := strings.TrimPrefix(moduleName, "std.") + "." + name
		exports[name] = FunctionValue(&Function{Name: name, Native: func(args []Value) (Value, error) {
			if len(args) != len(kinds) {
				return Null(), diagnostic.RuntimeCode(diagnostic.ArgumentCount, fmt.Errorf("std.%s expects %d arguments, got %d", qualified, len(kinds), len(args)))
			}
			for index, kind := range kinds {
				if kind == "number" && isNumeric(args[index]) {
					continue
				}
				if kind != "" && args[index].Kind != kind {
					return Null(), diagnostic.RuntimeCode(diagnostic.ArgumentType, fmt.Errorf("std.%s argument %d must be %s", qualified, index+1, kind))
				}
			}
			value, err := fn(args)
			if err != nil {
				return Null(), diagnostic.RuntimeCode(failureCode, fmt.Errorf("std.%s: %w", qualified, err))
			}
			return value, nil
		}})
	}
	switch moduleName {
	case "std.strings":
		for _, method := range []string{"trim", "trim_start", "trim_end", "upper", "lower", "reverse"} {
			m := method
			add(m, []Kind{StringKind}, func(a []Value) (Value, error) { v, _ := sequenceMethod(a[0], m, nil); return v, nil })
		}
		for _, method := range []string{"concat", "split", "contains", "starts_with", "ends_with", "index_of"} {
			m := method
			add(m, []Kind{StringKind, StringKind}, func(a []Value) (Value, error) { v, _ := sequenceMethod(a[0], m, a[1:]); return v, nil })
		}
		add("join", []Kind{ArrayKind, StringKind}, func(a []Value) (Value, error) { return joinValues(a[0].elements(), a[1].String), nil })
		add("replace", []Kind{StringKind, StringKind, StringKind}, func(a []Value) (Value, error) {
			return String(strings.ReplaceAll(a[0].String, a[1].String, a[2].String)), nil
		})
	case "std.files":
		addFileUtilities(add, resolve)
		add("read", []Kind{StringKind}, func(a []Value) (Value, error) { b, e := os.ReadFile(resolve(a[0].String)); return String(string(b)), e })
		add("write", []Kind{StringKind, StringKind}, func(a []Value) (Value, error) {
			return Null(), os.WriteFile(resolve(a[0].String), []byte(a[1].String), 0644)
		})
		add("append", []Kind{StringKind, StringKind}, func(a []Value) (Value, error) {
			f, e := os.OpenFile(resolve(a[0].String), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if e != nil {
				return Null(), e
			}
			_, e = f.WriteString(a[1].String)
			closeErr := f.Close()
			if e == nil {
				e = closeErr
			}
			return Null(), e
		})
		add("exists", []Kind{StringKind}, func(a []Value) (Value, error) {
			_, e := os.Stat(resolve(a[0].String))
			if os.IsNotExist(e) {
				return Boolean(false), nil
			}
			return Boolean(e == nil), e
		})
		add("mkdir", []Kind{StringKind}, func(a []Value) (Value, error) { return Null(), os.MkdirAll(resolve(a[0].String), 0755) })
		add("list", []Kind{StringKind}, func(a []Value) (Value, error) {
			entries, e := os.ReadDir(resolve(a[0].String))
			values := []Value{}
			for _, entry := range entries {
				values = append(values, String(entry.Name()))
			}
			return Array(values), e
		})
		add("join", []Kind{StringKind, StringKind}, func(a []Value) (Value, error) { return String(filepath.Join(a[0].String, a[1].String)), nil })
	case "std.time":
		add("now", nil, func([]Value) (Value, error) { return Integer(time.Now().UnixMilli()), nil })
		add("sleep", []Kind{IntegerKind}, func(a []Value) (Value, error) {
			if a[0].Integer < 0 || a[0].Integer > 86400000 {
				return Null(), fmt.Errorf("milliseconds must be between 0 and 86400000")
			}
			time.Sleep(time.Duration(a[0].Integer) * time.Millisecond)
			return Null(), nil
		})
		add("format", []Kind{IntegerKind}, func(a []Value) (Value, error) {
			return String(time.UnixMilli(a[0].Integer).UTC().Format(time.RFC3339Nano)), nil
		})
		add("parse", []Kind{StringKind}, func(a []Value) (Value, error) {
			t, e := time.Parse(time.RFC3339Nano, a[0].String)
			return Integer(t.UnixMilli()), e
		})
	case "std.math":
		exports["pi"], exports["e"] = Float(math.Pi), Float(math.E)
		for key, operation := range map[string]func(float64) float64{"abs": math.Abs, "sqrt": math.Sqrt, "floor": math.Floor, "ceil": math.Ceil, "round": math.Round, "sin": math.Sin, "cos": math.Cos, "log": math.Log} {
			fn := operation
			add(key, []Kind{"number"}, func(a []Value) (Value, error) { return finiteNumber(fn(a[0].number())) })
		}
		for key, operation := range map[string]func(float64, float64) float64{"pow": math.Pow, "min": math.Min, "max": math.Max} {
			fn := operation
			add(key, []Kind{"number", "number"}, func(a []Value) (Value, error) { return finiteNumber(fn(a[0].number(), a[1].number())) })
		}
	case "std.collections":
		add("length", []Kind{""}, func(a []Value) (Value, error) {
			if a[0].Kind == ObjectKind {
				return Integer(int64(len(a[0].Object))), nil
			}
			v := lengthOf(a[0])
			if v.Kind == NullKind {
				return v, fmt.Errorf("expected array, string, or object")
			}
			return v, nil
		})
		for _, method := range []string{"keys", "values"} {
			m := method
			add(m, []Kind{ObjectKind}, func(a []Value) (Value, error) {
				keys := []string{}
				for k := range a[0].Object {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				values := []Value{}
				for _, k := range keys {
					if m == "keys" {
						values = append(values, String(k))
					} else {
						values = append(values, a[0].Object[k])
					}
				}
				return Array(values), nil
			})
		}
		add("range", []Kind{IntegerKind}, func(a []Value) (Value, error) {
			n := a[0].Integer
			if n < 0 || n > 1000000 {
				return Null(), fmt.Errorf("count must be between 0 and 1000000")
			}
			values := make([]Value, int(n))
			for j := range values {
				values[j] = Integer(int64(j))
			}
			return Array(values), nil
		})
		add("contains", []Kind{ArrayKind, ""}, func(a []Value) (Value, error) { v, _ := sequenceMethod(a[0], "contains", a[1:]); return v, nil })
	case "std.serialization":
		addJSONUtilities(add, resolve)
		add("json_encode", []Kind{""}, encodeJSON)
		add("json_decode", []Kind{StringKind}, decodeJSON)
	case "std.networking":
		addNetworking(add, exports)
		add("get", []Kind{StringKind}, httpGet)
		add("post", []Kind{StringKind, StringKind, StringKind}, httpPost)
	case "std.security":
		addSecurity(add)
	case "std.os":
		addOS(add, exports)
	case "std.processes":
		add("run", []Kind{StringKind, ArrayKind}, runProcess)
		add("env", []Kind{StringKind}, func(a []Value) (Value, error) {
			value, ok := os.LookupEnv(a[0].String)
			if !ok {
				return Null(), nil
			}
			return String(value), nil
		})
		add("cwd", nil, func([]Value) (Value, error) { p, e := os.Getwd(); return String(p), e })
	default:
		return Null(), false
	}
	return Object(exports), true
}

func finiteNumber(n float64) (Value, error) {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return Null(), fmt.Errorf("result is outside the finite numeric domain")
	}
	return Float(n), nil
}
