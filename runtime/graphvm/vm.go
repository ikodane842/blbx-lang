// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"blbx_lang/runtime/graph"
	"blbx_lang/syntax/check"
	"blbx_lang/syntax/diagnostic"
	"blbx_lang/syntax/ir"
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

type signal int

const (
	noSignal signal = iota
	returnSignal
	assertSignal
	continueSignal
	breakSignal
)

type evalResult struct {
	value  Value
	signal signal
}

type Result struct {
	Output  []string         `json:"output"`
	Globals map[string]Value `json:"globals"`
	Last    Value            `json:"last"`
}

type moduleState int

const (
	moduleLoading moduleState = iota
	moduleLoaded
)

type moduleEntry struct {
	state moduleState
	value Value
}

type VM struct {
	// Trace observes component activation, not values. Workers have independent
	// VMs and do not inherit the callback, so callers need not synchronize it.
	Trace             func(*graph.Node)
	Visits            uint64
	ValueComputations uint64
	ValueCacheHits    uint64
	EffectSteps       uint64
	activation        *activation
	Args              []string
	ErrorWriter       io.Writer
	global            *Scope
	Output            []string
	Input             *bufio.Reader
	Writer            io.Writer
	moduleRoot        string
	currentFile       string
	modules           map[string]*moduleEntry
}

func (i *VM) visit(node *graph.Node) {
	i.Visits++
	if i.Trace != nil {
		i.Trace(node)
	}
}

func New() *VM {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	return &VM{
		ErrorWriter: os.Stderr,
		global:      NewScope(nil),
		Output:      []string{},
		Input:       bufio.NewReader(os.Stdin),
		Writer:      os.Stdout,
		moduleRoot:  wd,
		modules:     map[string]*moduleEntry{},
	}
}

func (i *VM) SetInput(input io.Reader) {
	if input == nil {
		i.Input = bufio.NewReader(os.Stdin)
		return
	}

	i.Input = bufio.NewReader(input)
}

func (i *VM) SetOutput(output io.Writer) {
	if output == nil {
		output = os.Stdout
	}
	i.Writer = output
}

func (i *VM) SetModuleRoot(root string) {
	if root == "" {
		return
	}

	abs, err := filepath.Abs(root)
	if err == nil {
		root = abs
	}

	i.moduleRoot = root
}

func (i *VM) SetCurrentFile(path string) {
	if path == "" {
		i.currentFile = ""
		return
	}

	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}

	i.currentFile = path
}

func (i *VM) ExecuteFile(path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, diagnostic.Diagnostic{Phase: "runtime", Code: diagnostic.SourceRead, Severity: "error", Message: err.Error(), Cause: err, File: path}
	}

	i.SetModuleRoot(filepath.Dir(path))
	i.SetCurrentFile(path)

	nodes, diagnostics := check.Source(path, string(data))
	if len(diagnostics) > 0 {
		failures := make([]error, len(diagnostics))
		for index := range diagnostics {
			failures[index] = diagnostics[index]
		}
		return Result{}, errors.Join(failures...)
	}
	return i.Execute(graph.Build(ir.Lower(nodes), path))
}

func (i *VM) Execute(program *graph.Program) (Result, error) {
	result, err := i.eval(program.Root, i.global)
	if err != nil {
		return Result{}, diagnostic.Runtime(err)
	}

	return Result{
		Output:  i.Output,
		Globals: i.global.Snapshot(),
		Last:    result.value,
	}, nil
}

func (i *VM) eval(node *graph.Node, scope *Scope) (evalResult, error) {
	if node == nil {
		return normal(Null())
	}
	i.visit(node)
	frame := i.activation
	if node.Pure && node.Type != graph.Literal && frame != nil {
		if value, ok := frame.values[node.ValueIdentity()]; ok {
			i.ValueCacheHits++
			return normal(value)
		}
	}
	result, err := i.evaluate(node, scope)
	if node.Pure && node.Type != graph.Literal && frame != nil && err == nil && result.signal == noSignal {
		frame.values[node.ValueIdentity()] = result.value
		i.ValueComputations++
	}
	return result, err
}

func (i *VM) evaluate(node *graph.Node, scope *Scope) (evalResult, error) {
	switch node.Type {
	case graph.ProgramRegion:
		return i.evalSequence(node, scope)
	case graph.ValueRef:
		return i.readVersion(node)
	case graph.Literal:
		return normal(literalValue(node))
	case graph.Identifier:
		return i.evalIdentifier(node, scope)
	case graph.Assign:
		return i.evalAssign(node, scope)
	case graph.Block:
		return i.evalBlock(node, NewScope(scope))
	case graph.Object:
		return i.evalObjectLiteral(node, NewScope(scope))
	case graph.Array:
		return i.evalArray(node, scope)
	case graph.Tuple:
		return i.evalTuple(node, scope)
	case graph.Function:
		return i.evalFunction(node, scope)
	case graph.Class:
		return i.evalClass(node, scope)
	case graph.Interface:
		contract := &Interface{Name: node.Name, Methods: map[string]int{}}
		for _, m := range node.Inputs {
			contract.Methods[m.Name] = len(m.Inputs)
		}
		value := Value{Kind: InterfaceKind, Interface: contract}
		scope.Define(node.Name, value)
		return normal(value)
	case graph.Return:
		return i.evalReturn(node, scope)
	case graph.Assert:
		return i.evalAssert(node, scope)
	case graph.Import:
		return i.evalImport(node, scope)
	case graph.MethodCall:
		return i.evalMethodCall(node, scope)
	case graph.ExpressionCall:
		return i.evalExpressionCall(node, scope)
	case graph.Call:
		return i.evalCall(node, scope)
	case graph.Member:
		return i.evalMember(node, scope)
	case graph.Index:
		return i.evalIndex(node, scope)
	case graph.Binary:
		return i.evalBinary(node, scope)
	case graph.If:
		return finishControl(i.evalIf(node, scope))
	case graph.For:
		return finishControl(i.evalFor(node, scope))
	case graph.While:
		return finishControl(i.evalWhile(node, scope))
	default:
		return normal(Null())
	}
}

func normal(value Value) (evalResult, error) {
	return evalResult{value: value, signal: noSignal}, nil
}

func literalValue(node *graph.Node) Value {
	state := node.Static
	switch state.Kind {
	case "integer":
		return Integer(state.Integer)
	case "float":
		return Float(state.Float)
	case "boolean":
		return Boolean(state.Boolean)
	case "null":
		return Null()
	default:
		return String(state.String)
	}
}

func (i *VM) evalIdentifier(node *graph.Node, scope *Scope) (evalResult, error) {
	if value, ok := scope.Get(node.Name); ok {
		return normal(value)
	}
	if node.Name == "self" {
		return evalResult{}, runtimeError(diagnostic.InvalidSelf, node, "self is only available inside an object or class instance")
	}

	if node.Name == "null" {
		return normal(Null())
	}
	return evalResult{}, runtimeError(diagnostic.UndefinedName, node, "undefined variable %q", node.Name)
}

func (i *VM) evalAssign(node *graph.Node, scope *Scope) (evalResult, error) {
	if len(node.Inputs) < 2 {
		return normal(Null())
	}

	target := node.Inputs[0]
	valueNode := node.Inputs[len(node.Inputs)-1]
	value, err := i.evalAssignmentValue(valueNode, scope)
	if err != nil {
		return value, err
	}
	if value.signal != noSignal {
		return value, nil
	}

	switch target.Type {
	case graph.ArrayPattern, graph.ObjectPattern:
		bindings := []patternBinding{}
		if err := collectPattern(target, value.value, &bindings); err != nil {
			return evalResult{}, err
		}
		for _, binding := range bindings {
			scope.Set(binding.name, binding.value)
		}
	case graph.Identifier:
		scope.Set(target.Name, value.value)
	case graph.Member:
		if err := i.assignMember(target, value.value, scope); err != nil {
			return value, err
		}
	case graph.Index:
		if err := i.assignIndex(target, value.value, scope); err != nil {
			return value, err
		}
	default:
		return normal(value.value)
	}

	if i.activation != nil && node.ValueType != "" {
		i.activation.bindings[node] = value.value
	}
	return normal(value.value)
}

func (i *VM) evalAssignmentValue(node *graph.Node, scope *Scope) (evalResult, error) {
	if node.Type == graph.Block && len(node.Inputs) == 0 {
		return normal(Object(map[string]Value{}))
	}

	return i.eval(node, scope)
}

func (i *VM) evalImport(node *graph.Node, scope *Scope) (evalResult, error) {
	module, err := i.loadModule(node.Name)
	if err != nil {
		return evalResult{}, err
	}

	if node.DataType == "from" {
		for _, imported := range node.Inputs {
			value, ok := module.Object[imported.Name]
			if !ok {
				return evalResult{}, runtimeError(diagnostic.MissingImportName, node, "module %q has no name %q", node.Name, imported.Name)
			}

			name := imported.Name
			if imported.Value != "" {
				name = imported.Value
			}

			scope.Set(name, value)
		}

		return normal(module)
	}

	if node.Value != "" {
		scope.Set(node.Value, module)
		return normal(module)
	}

	i.bindImportedModule(scope, strings.Split(node.Name, "."), module)
	return normal(module)
}

func (i *VM) bindImportedModule(scope *Scope, parts []string, module Value) {
	if len(parts) == 0 {
		return
	}

	if len(parts) == 1 {
		scope.Set(parts[0], module)
		return
	}

	rootName := parts[0]
	root, ok := scope.Get(rootName)
	if !ok || root.Kind != ObjectKind {
		root = Object(map[string]Value{})
	}

	current := root
	for _, part := range parts[1 : len(parts)-1] {
		next, ok := current.Object[part]
		if !ok || next.Kind != ObjectKind {
			next = Object(map[string]Value{})
			current.Object[part] = next
		}
		current = current.Object[part]
	}

	current.Object[parts[len(parts)-1]] = module
	scope.Set(rootName, root)
}

func (i *VM) loadModule(path string) (Value, error) {
	if module, ok := standardModuleAt(path, filepath.Dir(i.currentFile)); ok {
		return module, nil
	}
	modulePath, err := i.resolveModulePath(path)
	if err != nil {
		return Null(), err
	}

	entry, ok := i.modules[modulePath]
	if ok {
		return entry.value, nil
	}

	module := Object(map[string]Value{})
	entry = &moduleEntry{state: moduleLoading, value: module}
	i.modules[modulePath] = entry
	defer func() {
		if entry.state != moduleLoaded {
			delete(i.modules, modulePath)
		}
	}()

	data, err := os.ReadFile(modulePath)
	if err != nil {
		return Null(), diagnostic.RuntimeCode(diagnostic.SourceRead, err)
	}

	nodes, diagnostics := check.Source(modulePath, string(data))
	if len(diagnostics) > 0 {
		delete(i.modules, modulePath)
		return Null(), diagnostics[0]
	}
	program := graph.Build(ir.Lower(nodes), modulePath)
	moduleScope := NewScope(nil)

	previousFile := i.currentFile
	i.currentFile = modulePath
	result, err := i.eval(program.Root, moduleScope)
	i.currentFile = previousFile
	if err != nil {
		return result.value, err
	}

	for name, value := range moduleScope.Snapshot() {
		module.Object[name] = value
	}

	entry.state = moduleLoaded
	entry.value = module
	return module, nil
}

func (i *VM) resolveModulePath(path string) (string, error) {
	if path == "" {
		return "", runtimeError(diagnostic.ImportResolution, &graph.Node{}, "empty import path")
	}

	parts := strings.Split(path, ".")
	for _, part := range parts {
		if part == "" {
			return "", runtimeError(diagnostic.ImportResolution, &graph.Node{}, "invalid import path %q", path)
		}
	}

	root := i.moduleRoot
	if root == "" {
		root = "."
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	for index := 1; index < len(parts); index++ {
		initParts := append([]string{}, parts[:index]...)
		initParts = append(initParts, "__init__.bx")
		initPath := filepath.Join(append([]string{root}, initParts...)...)
		if _, err := os.Stat(initPath); err != nil {
			return "", runtimeError(diagnostic.ImportResolution, &graph.Node{}, "package %q requires %s", strings.Join(parts[:index], "."), initPath)
		}
	}

	filePath := filepath.Join(append([]string{root}, parts...)...) + ".bx"
	if _, err := os.Stat(filePath); err == nil {
		return filepath.Abs(filePath)
	}

	initParts := append([]string{}, parts...)
	initParts = append(initParts, "__init__.bx")
	initPath := filepath.Join(append([]string{root}, initParts...)...)
	if _, err := os.Stat(initPath); err == nil {
		return filepath.Abs(initPath)
	}

	return "", runtimeError(diagnostic.ImportResolution, &graph.Node{}, "could not resolve import %q from %s", path, root)
}

func (i *VM) assignMember(target *graph.Node, value Value, scope *Scope) error {
	if len(target.Inputs) < 2 {
		return nil
	}

	objectRef := target.Inputs[0]
	propertyRef := target.Inputs[1]

	if propertyRef.Type != graph.Identifier {
		return nil
	}
	if objectRef.Type != graph.Identifier {
		object, err := i.eval(objectRef, scope)
		if err != nil {
			return err
		}
		if object.value.Kind != ObjectKind {
			return runtimeError(diagnostic.InvalidReceiver, target, "member assignment requires an object")
		}
		object.value.Object[propertyRef.Name] = value
		return nil
	}

	object, ok := scope.Get(objectRef.Name)
	if objectRef.Name == "self" && (!ok || object.Kind != ObjectKind) {
		return runtimeError(diagnostic.InvalidSelf, target, "self is only available inside an object or class instance")
	}
	if !ok {
		return runtimeError(diagnostic.UndefinedName, target, "undefined variable %q", objectRef.Name)
	}
	if object.Kind != ObjectKind {
		return runtimeError(diagnostic.InvalidReceiver, target, "cannot assign member %q on %s", propertyRef.Name, object.Kind)
	}
	object.Object[propertyRef.Name] = value
	scope.Set(objectRef.Name, object)
	return nil
}

func (i *VM) evalBlock(node *graph.Node, scope *Scope) (evalResult, error) {
	return i.evalSequence(node, scope)
}

func (i *VM) evalObjectLiteral(node *graph.Node, scope *Scope) (evalResult, error) {
	object := map[string]Value{}
	scope = NewScope(scope)
	scope.Define("self", Object(object))

	for _, child := range node.Inputs {
		key := child.Inputs[0].Value
		value, err := i.eval(child.Inputs[len(child.Inputs)-1], scope)
		if err != nil {
			return value, err
		}
		if value.signal != noSignal {
			return value, nil
		}
		object[key] = value.value
	}

	return normal(Object(object))
}

func (i *VM) evalArray(node *graph.Node, scope *Scope) (evalResult, error) {
	values, result, err := i.evalArgs(node.ArgumentPlan, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}
	return normal(Array(values))
}

func (i *VM) evalTuple(node *graph.Node, scope *Scope) (evalResult, error) {
	result, err := i.evalArray(node, scope)
	result.value.Kind = TupleKind
	return result, err
}

func (i *VM) evalFunction(node *graph.Node, scope *Scope) (evalResult, error) {
	fn := &Function{
		Name:   node.Name,
		Body:   node.Body,
		Params: []Param{},
		Env:    scope,
	}

	for _, child := range node.Inputs {
		if child.Type == graph.Identifier {
			fn.Params = append(fn.Params, Param{
				Name:    child.Name,
				Default: Null(),
			})
			continue
		}

		if child.Type == graph.Tuple {
			for _, tupleChild := range child.Inputs {
				if tupleChild.Type == graph.Identifier {
					fn.Params = append(fn.Params, Param{
						Name:    tupleChild.Name,
						Default: Null(),
					})
				}
			}
			continue
		}

		if child.Type == graph.Assign && len(child.Inputs) >= 2 && child.Inputs[0].Type == graph.Identifier {
			defaultValue, err := i.eval(child.Inputs[1], scope)
			if err != nil {
				return defaultValue, err
			}
			fn.Params = append(fn.Params, Param{
				Name:    child.Inputs[0].Name,
				Default: defaultValue.value,
			})
			continue
		}

		if child.Type == graph.Block || child.Type == graph.Object {
			fn.Body = child
			continue
		}
	}

	return normal(FunctionValue(fn))
}

func (i *VM) evalReturn(node *graph.Node, scope *Scope) (evalResult, error) {
	if len(node.Inputs) == 0 {
		return evalResult{value: Null(), signal: returnSignal}, nil
	}

	value, err := i.eval(node.Inputs[0], scope)
	if err != nil || value.signal != noSignal {
		return value, err
	}
	value.signal = returnSignal
	return value, nil
}

func (i *VM) evalAssert(node *graph.Node, scope *Scope) (evalResult, error) {
	if len(node.Inputs) == 0 {
		return evalResult{value: Null(), signal: assertSignal}, nil
	}

	value, err := i.eval(node.Inputs[0], scope)
	if err != nil {
		return value, err
	}
	if value.signal != noSignal {
		return value, nil
	}

	return evalResult{value: value.value, signal: assertSignal}, nil
}

func (i *VM) evalCall(node *graph.Node, scope *Scope) (evalResult, error) {
	if node.Name == "super" && node.DataType == "direct-call" {
		proxy, ok := scope.Get("super")
		if !ok || proxy.Super == nil {
			return evalResult{}, runtimeError(diagnostic.InvalidSelf, node, "super requires a class method")
		}
		args, result, err := i.evalArgs(node.ArgumentPlan, scope)
		if err != nil || result.signal != noSignal {
			return result, err
		}
		constructor, err := i.superMember(proxy.Super, "", node)
		if err != nil {
			return evalResult{}, err
		}
		return i.callValue(constructor, args)
	}
	if node.DataType == "direct-call" && (node.Name == "try" || node.Name == "throw") {
		return i.exceptionCall(node, scope)
	}
	if node.Name == "typeof" && node.DataType == "direct-call" {
		if len(node.Inputs) != 1 {
			return evalResult{}, runtimeError(diagnostic.ArgumentCount, node, "typeof expects exactly one argument")
		}
		value, err := i.eval(node.Inputs[0], scope)
		if err != nil || value.signal != noSignal {
			return value, err
		}
		return normal(String(value.value.TypeName()))
	}
	if node.Name == "ord" && node.DataType == "direct-call" {
		if len(node.Inputs) != 1 {
			return evalResult{}, runtimeError(diagnostic.ArgumentCount, node, "ord expects exactly one argument")
		}
		value, err := i.eval(node.Inputs[0], scope)
		if err != nil || value.signal != noSignal {
			return value, err
		}
		if value.value.Kind != StringKind || !utf8.ValidString(value.value.String) || utf8.RuneCountInString(value.value.String) != 1 {
			return evalResult{}, runtimeError(diagnostic.ArgumentType, node, "ord expects a string containing exactly one Unicode code point")
		}
		codePoint, _ := utf8.DecodeRuneInString(value.value.String)
		return normal(Integer(int64(codePoint)))
	}
	if node.Name == "print" {
		return i.evalPrint(node, scope)
	}

	if node.Name == "input" {
		return i.evalInput(node, scope)
	}

	callee, ok := scope.Get(node.Name)
	if !ok {
		return evalResult{}, runtimeError(diagnostic.UndefinedName, node, "undefined function %q", node.Name)
	}
	if callee.Kind != FunctionKind && callee.Kind != ClassKind {
		return evalResult{}, runtimeError(diagnostic.NotCallable, node, "%q is %s, not callable", node.Name, callee.Kind)
	}

	args, result, err := i.evalArgs(node.ArgumentPlan, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	return i.callValue(callee, args)
}

func (i *VM) evalExpressionCall(node *graph.Node, scope *Scope) (evalResult, error) {
	callee, err := i.eval(node.Callee, scope)
	if err != nil {
		return callee, err
	}
	if callee.signal != noSignal {
		return callee, nil
	}

	if callee.value.Kind != FunctionKind && callee.value.Kind != ClassKind {
		return evalResult{}, runtimeError(diagnostic.NotCallable, node, "%s value is not callable", callee.value.Kind)
	}

	args, result, err := i.evalArgs(node.ArgumentPlan, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	return i.callValue(callee.value, args)
}

func (i *VM) evalPrint(node *graph.Node, scope *Scope) (evalResult, error) {
	args, result, err := i.evalArgs(node.ArgumentPlan, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	for _, arg := range args {
		text := arg.Display()
		i.Output = append(i.Output, text)
		fmt.Fprintln(i.Writer, text)
	}

	return normal(Null())
}

func (i *VM) evalInput(node *graph.Node, scope *Scope) (evalResult, error) {
	args, result, err := i.evalArgs(node.ArgumentPlan, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	if len(args) > 0 {
		fmt.Fprint(i.Writer, args[0].Display())
	}

	text, err := i.Input.ReadString('\n')
	if err != nil && err != io.EOF {
		return evalResult{}, runtimeError(diagnostic.InputFailure, node, "input failed: %v", err)
	}

	text = strings.TrimSuffix(text, "\n")
	text = strings.TrimSuffix(text, "\r")

	return normal(String(text))
}

func (i *VM) evalMethodCall(node *graph.Node, scope *Scope) (evalResult, error) {
	receiver, err := i.eval(node.Receiver, scope)
	if err != nil {
		return receiver, err
	}
	if receiver.signal != noSignal {
		return receiver, nil
	}

	method := node.Method
	if hook, ok := operatorHooks[method]; ok && receiver.value.Kind == ObjectKind {
		if _, exists := receiver.value.Object[method]; !exists {
			if _, exists = receiver.value.Object[hook]; exists {
				method = hook
			}
		}
	}
	if receiver.value.Super != nil {
		fn, err := i.superMember(receiver.value.Super, method, node)
		if err != nil {
			return evalResult{}, err
		}
		args, result, err := i.evalArgs(node.ArgumentPlan, scope)
		if err != nil || result.signal != noSignal {
			return result, err
		}
		return i.callValue(fn, args)
	}
	custom := false
	if receiver.value.Kind == ObjectKind {
		property, exists := receiver.value.Object[method]
		custom = exists && (property.Kind == FunctionKind || property.Kind == ClassKind)
	}
	if !custom {
		minimum, maximum, available := methodSignature(receiver.value, method)
		if !available {
			return evalResult{}, runtimeError(diagnostic.InvalidReceiver, node, "method %q is not available on %s", method, receiver.value.TypeName())
		}
		count := len(node.Arguments)
		if count < minimum || (maximum >= 0 && count > maximum) {
			return evalResult{}, runtimeError(diagnostic.ArgumentCount, node, "invalid argument count for %s.%s: got %d", receiver.value.TypeName(), method, count)
		}
	}
	// Boolean operators must decide whether to evaluate the RHS before the
	// ordinary eager argument evaluation used by other methods.
	if receiver.value.Kind == BooleanKind && (method == "and" || method == "or" || method == "not") {
		want := 1
		if method == "not" {
			want = 0
		}
		if len(node.Arguments) != want {
			return evalResult{}, runtimeError(diagnostic.ArgumentCount, node, "%s expects %d arguments", method, want)
		}
		if method == "not" {
			return normal(Boolean(!receiver.value.Boolean))
		}
		if method == "and" && !receiver.value.Boolean {
			return normal(Boolean(false))
		}
		if method == "or" && receiver.value.Boolean {
			return normal(Boolean(true))
		}
		right, err := i.eval(node.Arguments[0], scope)
		if err != nil || right.signal != noSignal {
			return right, err
		}
		if right.value.Kind != BooleanKind {
			return evalResult{}, runtimeError(diagnostic.ArgumentType, node, "%s requires a boolean argument", method)
		}
		return normal(right.value)
	}
	args, result, err := i.evalArgs(node.ArgumentPlan, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	if method == "extend" && !custom {
		if args[0].Kind != ArrayKind {
			return evalResult{}, runtimeError(diagnostic.ArgumentType, node, "extend expects an array argument")
		}
		*receiver.value.array = append(receiver.value.elements(), args[0].elements()...)
		return normal(Null())
	}

	if method == "in" && !custom {
		if args[0].Kind != ObjectKind {
			return evalResult{}, runtimeError(diagnostic.ArgumentType, node, "in expects an object argument")
		}
		_, exists := args[0].Object[receiver.value.String]
		return normal(Boolean(exists))
	}

	if method == "continue" && !custom {
		return evalResult{value: Null(), signal: continueSignal}, nil
	}

	if method == "break" && !custom {
		return evalResult{value: Null(), signal: breakSignal}, nil
	}

	if receiver.value.Kind == ObjectKind {
		if property, ok := receiver.value.Object[method]; ok {
			if property.Kind != FunctionKind && property.Kind != ClassKind {
				if method == "elem" || method == "idx" || method == "index" || method == "step" {
					return normal(i.callMethod(receiver.value, method, args))
				}
				return evalResult{}, runtimeError(diagnostic.NotCallable, node, "member %q is %s, not callable", method, property.Kind)
			}
			return i.callValue(bindReceiver(property, receiver.value), args)
		}
	}

	if receiver.value.Kind == IntegerKind {
		if value, handled, err := bitwiseMethod(receiver.value.Integer, method, args, node); handled {
			return evalResult{value: value}, err
		}
	}
	return normal(i.callMethod(receiver.value, method, args))
}

func (i *VM) callFunction(fn *Function, args []Value) (evalResult, error) {
	return i.callFunctionBody(fn, args, false)
}

// Control-flow callbacks preserve return/assert signals for their enclosing
// statement; ordinary calls remain function-return boundaries.
func (i *VM) callFunctionBody(fn *Function, args []Value, control bool) (evalResult, error) {
	if fn == nil {
		return normal(Null())
	}
	if fn.NativeContext != nil {
		value, err := fn.NativeContext(i, args)
		return evalResult{value: value}, err
	}
	if fn.Native != nil {
		value, err := fn.Native(args)
		return evalResult{value: value}, err
	}

	callScope := NewScope(fn.Env)
	for index, param := range fn.Params {
		if param.Name == "self" {
			return evalResult{}, runtimeError(diagnostic.InvalidSelf, fn.Body, "self cannot be used as a parameter")
		}
		value := param.Default
		if index < len(args) {
			value = args[index]
		}
		callScope.Define(param.Name, value)
	}

	result, err := i.eval(fn.Body, callScope)
	if err != nil {
		return result, err
	}

	if !control && result.signal == returnSignal {
		return normal(result.value)
	}

	return result, nil
}

func (i *VM) callMethod(receiver Value, method string, args []Value) Value {
	if method == "to_bool" {
		return Boolean(receiver.IsTruthy())
	}
	if (method == "indexes" || method == "values") && receiver.Kind == ObjectKind && receiver.Class == nil && len(args) == 0 {
		return objectEntries(receiver, method)
	}
	if result, handled := sequenceMethod(receiver, method, args); handled {
		return result
	}
	switch method {
	case "to_str":
		if len(args) == 0 {
			return String(receiver.Display())
		}
	case "to_int":
		if len(args) != 0 {
			return Null()
		}
		switch receiver.Kind {
		case IntegerKind:
			return receiver
		case FloatKind:
			if !math.IsNaN(receiver.Float) && receiver.Float >= -0x1p63 && receiver.Float < 0x1p63 {
				return Integer(int64(receiver.Float))
			}
		case StringKind:
			value, err := strconv.ParseInt(receiver.String, 10, 64)
			if err == nil {
				return Integer(value)
			}
		}
	case "to_float":
		if len(args) != 0 {
			return Null()
		}
		switch receiver.Kind {
		case IntegerKind:
			return Float(float64(receiver.Integer))
		case FloatKind:
			return receiver
		case StringKind:
			value, err := strconv.ParseFloat(receiver.String, 64)
			if err == nil && !math.IsNaN(value) && !math.IsInf(value, 0) {
				return Float(value)
			}
		}
	case "add":
		if len(args) > 0 {
			return numericOperation(receiver, args[0], "+")
		}
	case "sub":
		if len(args) > 0 {
			return numericOperation(receiver, args[0], "-")
		}
	case "mul":
		if len(args) > 0 {
			return numericOperation(receiver, args[0], "*")
		}
	case "div":
		if len(args) > 0 {
			return numericOperation(receiver, args[0], "/")
		}
	case "mod":
		if len(args) > 0 {
			return numericOperation(receiver, args[0], "%")
		}
	case "gt":
		if len(args) > 0 {
			return Boolean(compareValues(receiver, args[0], ">"))
		}
	case "gte":
		if len(args) > 0 {
			return Boolean(compareValues(receiver, args[0], ">="))
		}
	case "lt":
		if len(args) > 0 {
			return Boolean(compareValues(receiver, args[0], "<"))
		}
	case "lte":
		if len(args) > 0 {
			return Boolean(compareValues(receiver, args[0], "<="))
		}
	case "eq":
		if len(args) > 0 {
			return Boolean(receiver.Equal(args[0]))
		}
	case "neq":
		if len(args) > 0 {
			return Boolean(!receiver.Equal(args[0]))
		}
	case "not":
		if receiver.Kind == BooleanKind && len(args) == 0 {
			return Boolean(!receiver.Boolean)
		}
	case "type", "typeof":
		return String(receiver.TypeName())
	case "length":
		return lengthOf(receiver)
	case "elem":
		if receiver.Kind == ObjectKind {
			if value, ok := receiver.Object["elem"]; ok {
				return value
			}
		}
	case "idx", "index":
		if receiver.Kind == ObjectKind {
			if value, ok := receiver.Object["idx"]; ok {
				return value
			}
		}
	case "step":
		if receiver.Kind == ObjectKind {
			if value, ok := receiver.Object["step"]; ok {
				return value
			}
		}
	case "continue":
		return Null()
	case "break":
		return Null()
	}

	return Null()
}

func (i *VM) evalMember(node *graph.Node, scope *Scope) (evalResult, error) {
	if len(node.Inputs) < 2 {
		return normal(Null())
	}

	receiver, err := i.eval(node.Inputs[0], scope)
	if err != nil {
		return receiver, err
	}
	if receiver.signal != noSignal {
		return receiver, nil
	}

	property := node.Inputs[1].Name
	if receiver.value.Super != nil {
		v, err := i.superMember(receiver.value.Super, property, node)
		return evalResult{value: v}, err
	}
	if (property == "indexes" || property == "values") && receiver.value.Kind == ObjectKind && receiver.value.Class == nil {
		return normal(objectEntries(receiver.value, property))
	}
	if property == "typeof" {
		return normal(String(receiver.value.TypeName()))
	}
	if property == "length" {
		if receiver.value.Kind == ArrayKind || receiver.value.Kind == TupleKind || receiver.value.Kind == StringKind {
			return normal(lengthOf(receiver.value))
		}
	}

	if receiver.value.Kind == ObjectKind {
		if value, ok := receiver.value.Object[property]; ok {
			return normal(bindReceiver(value, receiver.value))
		}
	}

	return evalResult{}, runtimeError(diagnostic.MissingMember, node, "undefined member %q on %s", property, receiver.value.TypeName())
}

func (i *VM) evalIndex(node *graph.Node, scope *Scope) (evalResult, error) {
	if len(node.Inputs) < 2 {
		return normal(Null())
	}

	target, err := i.eval(node.Inputs[0], scope)
	if err != nil {
		return target, err
	}
	index, err := i.eval(node.Inputs[1], scope)
	if err != nil {
		return index, err
	}

	if target.value.Kind == ObjectKind && index.value.Kind == StringKind {
		if value, ok := target.value.Object[index.value.String]; ok {
			return normal(bindReceiver(value, target.value))
		}

		return evalResult{}, runtimeError(diagnostic.MissingMember, node, "undefined member %q on object", index.value.String)
	}

	if target.value.Kind == StringKind && index.value.Kind == IntegerKind {
		runes := []rune(target.value.String)
		n := index.value.Integer
		if n < 0 {
			n += int64(len(runes))
		}
		if n < 0 || n >= int64(len(runes)) {
			return normal(Null())
		}
		return normal(String(string(runes[n])))
	}
	if (target.value.Kind == ArrayKind || target.value.Kind == TupleKind) && index.value.Kind == IntegerKind {
		arrayIndex := int(index.value.Integer)
		if arrayIndex < 0 || arrayIndex >= len(target.value.elements()) {
			return normal(Null())
		}

		return normal(target.value.elements()[arrayIndex])
	}

	return normal(Null())
}

func (i *VM) evalBinary(node *graph.Node, scope *Scope) (evalResult, error) {
	if len(node.Inputs) < 2 {
		return normal(Null())
	}

	values, result, err := i.evalArgs(node.ArgumentPlan, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}
	left, right := values[0], values[1]

	switch node.Name {
	case "*":
		if method, ok := operatorMethod(left, "mul"); ok {
			if method.Kind != FunctionKind {
				return evalResult{}, runtimeError(diagnostic.NotCallable, node, "multiplication overload must be a function")
			}
			return i.callValue(bindReceiver(method, left), []Value{right})
		}
		if !isNumeric(left) || !isNumeric(right) {
			return evalResult{}, runtimeError(diagnostic.ArgumentType, node, "multiplication requires numbers or a left-hand __mul__ overload")
		}
		if left.Kind == FloatKind || right.Kind == FloatKind {
			return normal(Float(left.number() * right.number()))
		}
		return normal(Integer(left.Integer * right.Integer))
	default:
		return normal(Null())
	}
}

func (i *VM) evalIf(node *graph.Node, scope *Scope) (evalResult, error) {
	if node.Condition == nil {
		return normal(Null())
	}
	condition, err := i.eval(node.Condition, scope)
	if err != nil || condition.signal != noSignal {
		return condition, err
	}
	branch := node.Else
	if condition.value.IsTruthy() {
		branch = node.Then
	}
	if branch == nil {
		return normal(Null())
	}
	return i.evalScopedValue(branch, scope)
}

func finishControl(result evalResult, err error) (evalResult, error) {
	if err == nil && result.signal == assertSignal {
		result.signal = noSignal
	}
	return result, err
}

func (i *VM) evalFor(node *graph.Node, scope *Scope) (evalResult, error) {
	if len(node.Inputs) < 2 {
		return normal(Null())
	}

	iterable, err := i.eval(node.Iterable, scope)
	if err != nil || iterable.signal != noSignal {
		return iterable, err
	}
	callback, err := i.eval(node.Body, scope)
	if err != nil || callback.signal != noSignal {
		return callback, err
	}

	if callback.value.Kind != FunctionKind {
		return evalResult{}, runtimeError(diagnostic.ArgumentType, node, "for callback must be a function")
	}
	next, err := i.iterator(iterable.value, node)
	if err != nil {
		return evalResult{}, err
	}

	last := Null()
	step := int64(1)
	for gate, index := node.Entry, 0; gate != nil; gate, index = gate.Repeat, index+1 {
		i.visit(gate)
		elem, done, err := next()
		if err != nil {
			return evalResult{}, err
		}
		if done {
			break
		}
		loopValue := Object(map[string]Value{
			"elem": elem,
			"idx":  Integer(int64(index)),
			"step": Integer(step),
		})
		loopValue.Cursor = true
		result, err := i.callFunctionBody(callback.value.Function, []Value{loopValue}, true)
		if err != nil {
			return result, err
		}
		last = result.value
		if result.signal == continueSignal {
			step++
			continue
		}
		if result.signal == breakSignal {
			break
		}
		if result.signal == assertSignal {
			return normal(result.value)
		}
		if result.signal == returnSignal {
			return result, nil
		}
		step++
	}

	return normal(last)
}

func (i *VM) evalWhile(node *graph.Node, scope *Scope) (evalResult, error) {
	if len(node.Inputs) < 2 {
		return normal(Null())
	}

	last := Null()
	for gate, index := node.Entry, int64(0); gate != nil; gate, index = gate.Repeat, index+1 {
		i.visit(gate)
		condition, err := i.eval(gate.Condition, scope)
		if err != nil || condition.signal != noSignal {
			return condition, err
		}
		if !condition.value.IsTruthy() {
			break
		}

		cursor := Object(map[string]Value{
			"elem": Null(),
			"idx":  Integer(index),
			"step": Integer(index + 1),
		})
		cursor.Cursor = true
		result, err := i.evalScopedValue(gate.Body, scope, cursor)
		if err != nil {
			return result, err
		}
		last = result.value
		if result.signal == continueSignal {
			continue
		}
		if result.signal == breakSignal {
			break
		}
		if result.signal == assertSignal {
			return normal(result.value)
		}
		if result.signal == returnSignal {
			return result, nil
		}
	}

	return normal(last)
}

func (i *VM) evalScopedValue(node *graph.Node, scope *Scope, args ...Value) (evalResult, error) {
	result, err := i.eval(node, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	if result.value.Kind == FunctionKind {
		return i.callFunctionBody(result.value.Function, args, true)
	}

	return result, nil
}

func runtimeError(code string, node *graph.Node, format string, args ...interface{}) error {
	return diagnostic.Diagnostic{Phase: "runtime", Code: code, Severity: "error", Line: node.Line, Message: fmt.Sprintf(format, args...)}
}

func lengthOf(value Value) Value {
	switch value.Kind {
	case ArrayKind, TupleKind:
		return Integer(int64(len(value.elements())))
	case StringKind:
		return Integer(int64(len([]rune(value.String))))
	default:
		return Null()
	}
}

func numericOperation(left Value, right Value, operator string) Value {
	if !isNumeric(left) || !isNumeric(right) {
		return Null()
	}

	if operator == "/" {
		if right.number() == 0 {
			return Null()
		}
		return Float(left.number() / right.number())
	}

	if operator == "%" {
		if left.Kind != IntegerKind || right.Kind != IntegerKind || right.Integer == 0 {
			return Null()
		}
		return Integer(left.Integer % right.Integer)
	}

	if left.Kind == IntegerKind && right.Kind == IntegerKind {
		switch operator {
		case "+":
			return Integer(left.Integer + right.Integer)
		case "-":
			return Integer(left.Integer - right.Integer)
		case "*":
			return Integer(left.Integer * right.Integer)
		}
	}

	switch operator {
	case "+":
		return Float(left.number() + right.number())
	case "-":
		return Float(left.number() - right.number())
	case "*":
		return Float(left.number() * right.number())
	default:
		return Null()
	}
}

func compareValues(left Value, right Value, operator string) bool {
	if isNumeric(left) && isNumeric(right) {
		switch operator {
		case ">":
			return left.number() > right.number()
		case ">=":
			return left.number() >= right.number()
		case "<":
			return left.number() < right.number()
		case "<=":
			return left.number() <= right.number()
		}
	}

	if left.Kind == StringKind && right.Kind == StringKind {
		switch operator {
		case ">":
			return left.String > right.String
		case ">=":
			return left.String >= right.String
		case "<":
			return left.String < right.String
		case "<=":
			return left.String <= right.String
		}
	}

	return false
}
