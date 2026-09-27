package graphvm

import (
	"blbx_lang/runtime/graph"
	"blbx_lang/syntax/diagnostic"
)

// An activation owns values and effect completions for one region invocation.
// Reentering a body never reuses another invocation's versions or effect tokens.
type activation struct {
	values    map[*graph.Node]Value
	bindings  map[*graph.Node]Value
	completed map[*graph.Node]completion
}

type completion struct {
	result evalResult
	err    error
}

func (i *VM) evalSequence(region *graph.Node, scope *Scope) (evalResult, error) {
	previous := i.activation
	i.activation = &activation{values: map[*graph.Node]Value{}, bindings: map[*graph.Node]Value{}, completed: map[*graph.Node]completion{}}
	defer func() { i.activation = previous }()
	return i.demandEffect(region.Exit, scope)
}

// demandEffect resolves prerequisite tokens before their consumer. The explicit
// stack bounds host stack use for long statement chains. Next edges are useful
// for inspection but are not the scheduler: execution follows EffectIn.
func (i *VM) demandEffect(target *graph.Node, scope *Scope) (evalResult, error) {
	if target == nil {
		return normal(Null())
	}
	path := []*graph.Node{}
	for step := target; step != nil; step = step.EffectIn {
		if done, ok := i.activation.completed[step]; ok {
			if done.err != nil || done.result.signal != noSignal {
				return done.result, done.err
			}
			break
		}
		path = append(path, step)
	}
	for index := len(path) - 1; index >= 0; index-- {
		step := path[index]
		i.visit(step)
		i.EffectSteps++
		result, err := i.eval(step.Inputs[0], scope)
		i.activation.completed[step] = completion{result, err}
		if err != nil || result.signal != noSignal {
			return result, err
		}
	}
	done := i.activation.completed[target]
	return done.result, done.err
}

func (i *VM) readVersion(node *graph.Node) (evalResult, error) {
	if i.activation != nil {
		if value, ok := i.activation.bindings[node.Definition]; ok {
			return normal(value)
		}
	}
	// Never silently reread the name or rerun the defining assignment.
	return evalResult{}, runtimeError(diagnostic.NativeFailure, node, "value version for %q was demanded before its defining effect completed", node.Name)
}

// Each argument demand gets fresh effect tokens, but retains the enclosing
// activation's immutable value cache and completed binding versions. In
// particular, loop conditions must execute their effectful calls on every visit.
func (i *VM) evalArgs(region *graph.Node, scope *Scope) ([]Value, evalResult, error) {
	path := []*graph.Node{}
	if region != nil {
		for step := region.Exit; step != nil; step = step.EffectIn {
			path = append(path, step)
		}
	}
	values := make([]Value, 0, len(path))
	for index := len(path) - 1; index >= 0; index-- {
		step := path[index]
		i.visit(step)
		i.EffectSteps++
		result, err := i.eval(step.Inputs[0], scope)
		if err != nil || result.signal != noSignal {
			return values, result, err
		}
		values = append(values, result.value)
	}
	return values, evalResult{}, nil
}
