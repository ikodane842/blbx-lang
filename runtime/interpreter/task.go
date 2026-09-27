package interpreter

import (
	"blbx_lang/syntax/diagnostic"
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Task publishes its immutable result by closing done. BLBX state never crosses
// the worker boundary without a snapshot; only task handles are shared.
type Task struct {
	done        chan struct{}
	result      Value
	err         error
	output      string
	errorOutput string
	lines       []string
	printed     sync.Once
}

func taskModule() Value {
	exports := map[string]Value{}
	exports["spawn"] = FunctionValue(&Function{Name: "task.spawn", NativeContext: func(i *Interpreter, args []Value) (Value, error) {
		if len(args) == 0 {
			return Null(), diagnostic.RuntimeCode(diagnostic.ArgumentCount, fmt.Errorf("std.task.spawn expects a function followed by its arguments"))
		}
		if args[0].Kind != FunctionKind {
			return Null(), diagnostic.RuntimeCode(diagnostic.ArgumentType, fmt.Errorf("std.task.spawn expects a function followed by its arguments"))
		}
		c := newSnapshot()
		fn := c.value(args[0])
		copied := make([]Value, len(args)-1)
		for n, v := range args[1:] {
			copied[n] = c.value(v)
		}
		worker := New()
		worker.Args = append([]string(nil), i.Args...)
		worker.global = c.scope(i.global)
		worker.moduleRoot, worker.currentFile = i.moduleRoot, i.currentFile
		worker.SetInput(strings.NewReader(""))
		var output bytes.Buffer
		var errorOutput bytes.Buffer
		worker.ErrorWriter = &errorOutput
		worker.SetOutput(&output)
		task := &Task{done: make(chan struct{})}
		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					task.err = diagnostic.RuntimeCode(diagnostic.TaskFailure, fmt.Errorf("task failed: %v", recovered))
				}
				task.output, task.lines = output.String(), worker.Output
				task.errorOutput = errorOutput.String()
				close(task.done)
			}()
			result, err := worker.callFunction(fn.Function, copied)
			task.result, task.err = result.value, err
		}()
		return Value{Kind: TaskKind, Task: task}, nil
	}})
	for _, name := range []string{"await", "done"} {
		operation := name
		exports[name] = FunctionValue(&Function{Name: "task." + name, NativeContext: func(i *Interpreter, args []Value) (Value, error) {
			if len(args) != 1 {
				return Null(), diagnostic.RuntimeCode(diagnostic.ArgumentCount, fmt.Errorf("std.task.%s expects one task handle", operation))
			}
			if args[0].Kind != TaskKind || args[0].Task == nil {
				return Null(), diagnostic.RuntimeCode(diagnostic.ArgumentType, fmt.Errorf("std.task.%s expects one task handle", operation))
			}
			task := args[0].Task
			if operation == "done" {
				select {
				case <-task.done:
					return Boolean(true), nil
				default:
					return Boolean(false), nil
				}
			}
			<-task.done
			task.printed.Do(func() {
				fmt.Fprint(i.Writer, task.output)
				fmt.Fprint(i.ErrorWriter, task.errorOutput)
				i.Output = append(i.Output, task.lines...)
			})
			if task.err != nil {
				return Null(), fmt.Errorf("std.task.await: %w", task.err)
			}
			return newSnapshot().value(task.result), nil
		}})
	}
	return Object(exports)
}

// Memoizing every mutable graph node preserves cycles and aliases, including
// recursive functions, self references, and arrays containing their owner.
type snapshot struct {
	layers    map[uintptr]map[*Class]map[string]Value
	supers    map[*superReference]*superReference
	scopes    map[*Scope]*Scope
	functions map[*Function]*Function
	classes   map[*Class]*Class
	objects   map[uintptr]map[string]Value
	arrays    map[*[]Value]*[]Value
}

func newSnapshot() *snapshot {
	return &snapshot{scopes: map[*Scope]*Scope{}, functions: map[*Function]*Function{}, classes: map[*Class]*Class{}, objects: map[uintptr]map[string]Value{}, arrays: map[*[]Value]*[]Value{}, layers: map[uintptr]map[*Class]map[string]Value{}, supers: map[*superReference]*superReference{}}
}
func (c *snapshot) scope(s *Scope) *Scope {
	if s == nil {
		return nil
	}
	if copy, ok := c.scopes[s]; ok {
		return copy
	}
	copy := NewScope(nil)
	c.scopes[s] = copy
	copy.parent = c.scope(s.parent)
	for k, v := range s.values {
		copy.values[k] = c.value(v)
	}
	return copy
}
func (c *snapshot) class(s *Class) *Class {
	if s == nil {
		return nil
	}
	if copy, ok := c.classes[s]; ok {
		return copy
	}
	copy := *s
	c.classes[s] = &copy
	copy.Parent, copy.Env = c.class(s.Parent), c.scope(s.Env)
	copy.Parents = nil
	for _, p := range s.Parents {
		copy.Parents = append(copy.Parents, c.class(p))
	}
	copy.MRO = nil
	for _, p := range s.MRO {
		copy.MRO = append(copy.MRO, c.class(p))
	}
	return &copy
}
func (c *snapshot) value(v Value) Value {
	if v.Layers != nil {
		key := reflect.ValueOf(v.Layers).Pointer()
		if copy, ok := c.layers[key]; ok {
			v.Layers = copy
		} else {
			original := v.Layers
			v.Layers = map[*Class]map[string]Value{}
			c.layers[key] = v.Layers
			for class, fields := range original {
				v.Layers[c.class(class)] = c.value(Object(fields)).Object
			}
		}
	}
	if v.Super != nil {
		original := v.Super
		if copy, ok := c.supers[original]; ok {
			v.Super = copy
		} else {
			copy := &superReference{}
			c.supers[original] = copy
			v.Super = copy
			copy.Owner = c.class(original.Owner)
			copy.Receiver = c.value(original.Receiver)
		}
	}
	if v.Object != nil {
		key := reflect.ValueOf(v.Object).Pointer()
		if copy, ok := c.objects[key]; ok {
			v.Object = copy
		} else {
			original := v.Object
			v.Object = map[string]Value{}
			c.objects[key] = v.Object
			for k, item := range original {
				v.Object[k] = c.value(item)
			}
		}
	}
	if v.array != nil {
		key := v.array
		if copy, ok := c.arrays[key]; ok {
			v.array = copy
		} else {
			original := v.elements()
			copy := make([]Value, len(original))
			v.array = &copy
			c.arrays[key] = v.array
			for n, item := range original {
				copy[n] = c.value(item)
			}
		}
	}
	if v.Function != nil {
		original := v.Function
		if copy, ok := c.functions[original]; ok {
			v.Function = copy
		} else {
			copy := *original
			v.Function = &copy
			c.functions[original] = &copy
			copy.Env = c.scope(original.Env)
			copy.Owner = c.class(original.Owner)
			copy.Params = append([]Param(nil), original.Params...)
			for n := range copy.Params {
				copy.Params[n].Default = c.value(copy.Params[n].Default)
			}
		}
	}
	v.Class = c.class(v.Class)
	return v
}
