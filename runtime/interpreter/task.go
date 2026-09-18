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
	done    chan struct{}
	result  Value
	err     error
	output  string
	lines   []string
	printed sync.Once
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
		worker.global = c.scope(i.global)
		worker.moduleRoot, worker.currentFile = i.moduleRoot, i.currentFile
		worker.SetInput(strings.NewReader(""))
		var output bytes.Buffer
		worker.SetOutput(&output)
		task := &Task{done: make(chan struct{})}
		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					task.err = diagnostic.RuntimeCode(diagnostic.TaskFailure, fmt.Errorf("task failed: %v", recovered))
				}
				task.output, task.lines = output.String(), worker.Output
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
			task.printed.Do(func() { fmt.Fprint(i.Writer, task.output); i.Output = append(i.Output, task.lines...) })
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
type arrayIdentity struct {
	pointer uintptr
	length  int
}
type snapshot struct {
	scopes    map[*Scope]*Scope
	functions map[*Function]*Function
	classes   map[*Class]*Class
	objects   map[uintptr]map[string]Value
	arrays    map[arrayIdentity][]Value
}

func newSnapshot() *snapshot {
	return &snapshot{map[*Scope]*Scope{}, map[*Function]*Function{}, map[*Class]*Class{}, map[uintptr]map[string]Value{}, map[arrayIdentity][]Value{}}
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
	return &copy
}
func (c *snapshot) value(v Value) Value {
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
	if v.Array != nil {
		key := arrayIdentity{reflect.ValueOf(v.Array).Pointer(), len(v.Array)}
		if copy, ok := c.arrays[key]; ok {
			v.Array = copy
		} else {
			original := v.Array
			v.Array = make([]Value, len(original))
			c.arrays[key] = v.Array
			for n, item := range original {
				v.Array[n] = c.value(item)
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
			copy.Params = append([]Param(nil), original.Params...)
			for n := range copy.Params {
				copy.Params[n].Default = c.value(copy.Params[n].Default)
			}
		}
	}
	v.Class = c.class(v.Class)
	return v
}
