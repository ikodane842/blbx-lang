package interpreter

import (
	"blbx_lang/syntax/diagnostic"
	"blbx_lang/syntax/ir"
	"errors"
)

type thrownValue struct {
	value      Value
	diagnostic diagnostic.Diagnostic
}

func (e *thrownValue) Error() string { return e.diagnostic.Error() }
func (e *thrownValue) Unwrap() error { return e.diagnostic }

func (i *Interpreter) exceptionCall(node ir.Node, scope *Scope) (evalResult, error) {
	count := 1
	if node.Name == "try" {
		count = 2
	}
	if len(node.Children) != count {
		return evalResult{}, runtimeError(diagnostic.ArgumentCount, node, "%s expects %d arguments", node.Name, count)
	}
	args, result, err := i.evalArgs(node.Children, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}
	if node.Name == "throw" {
		return evalResult{}, &thrownValue{value: args[0], diagnostic: diagnostic.Diagnostic{Phase: "runtime", Code: diagnostic.UserThrown, Severity: "error", Line: node.Line, Message: args[0].Display()}}
	}
	if args[0].Kind != FunctionKind || args[1].Kind != FunctionKind {
		return evalResult{}, runtimeError(diagnostic.ArgumentType, node, "try expects work and catch functions")
	}
	result, err = i.callValue(args[0], nil)
	if err == nil {
		return result, nil
	}
	normalized := diagnostic.Runtime(err)
	var d diagnostic.Diagnostic
	errors.As(normalized, &d)
	if d.Phase == "" {
		d.Phase = "syntax"
	}
	payload := Null()
	var thrown *thrownValue
	if errors.As(err, &thrown) {
		payload = newSnapshot().value(thrown.value)
	}
	errorValue := Object(map[string]Value{"code": String(d.Code), "message": String(d.Message), "phase": String(d.Phase), "value": payload})
	return i.callValue(args[1], []Value{errorValue})
}
