package interpreter

import (
	"blbx_lang/syntax/ir"
	"blbx_lang/syntax/lexer"
	"blbx_lang/syntax/parser"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

type Interpreter struct {
	global      *Scope
	Output      []string
	Input       *bufio.Reader
	moduleRoot  string
	currentFile string
	modules     map[string]*moduleEntry
}

func New() *Interpreter {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	return &Interpreter{
		global:     NewScope(nil),
		Output:     []string{},
		Input:      bufio.NewReader(os.Stdin),
		moduleRoot: wd,
		modules:    map[string]*moduleEntry{},
	}
}

func (i *Interpreter) SetInput(input io.Reader) {
	if input == nil {
		i.Input = bufio.NewReader(os.Stdin)
		return
	}

	i.Input = bufio.NewReader(input)
}

func (i *Interpreter) SetModuleRoot(root string) {
	if root == "" {
		return
	}

	abs, err := filepath.Abs(root)
	if err == nil {
		root = abs
	}

	i.moduleRoot = root
}

func (i *Interpreter) SetCurrentFile(path string) {
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

func (i *Interpreter) ExecuteFile(path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}

	i.SetModuleRoot(filepath.Dir(path))
	i.SetCurrentFile(path)

	var l lexer.Lexer
	l.Set(string(data))
	l.Tokenize()

	var p parser.Parser
	p.Set(l.Get())
	p.Parse()

	return i.Execute(ir.Lower(p.Get()))
}

func (i *Interpreter) Execute(program ir.Node) (Result, error) {
	result, err := i.eval(program, i.global)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Output:  i.Output,
		Globals: i.global.Snapshot(),
		Last:    result.value,
	}, nil
}

func (i *Interpreter) eval(node ir.Node, scope *Scope) (evalResult, error) {
	switch node.Type {
	case ir.Program:
		return i.evalSequence(node.Children, scope)
	case ir.Literal:
		return normal(literalValue(node))
	case ir.Identifier:
		return i.evalIdentifier(node, scope)
	case ir.Assign:
		return i.evalAssign(node, scope)
	case ir.Block:
		return i.evalBlock(node, NewScope(scope))
	case ir.Array:
		return i.evalArray(node, scope)
	case ir.Tuple:
		return i.evalTuple(node, scope)
	case ir.Function:
		return i.evalFunction(node, scope)
	case ir.Return:
		return i.evalReturn(node, scope)
	case ir.Assert:
		return i.evalAssert(node, scope)
	case ir.Import:
		return i.evalImport(node, scope)
	case ir.Call:
		return i.evalCall(node, scope)
	case ir.Member:
		return i.evalMember(node, scope)
	case ir.Index:
		return i.evalIndex(node, scope)
	case ir.Binary:
		return i.evalBinary(node, scope)
	case ir.If:
		return i.evalIf(node, scope)
	case ir.For:
		return i.evalFor(node, scope)
	case ir.While:
		return i.evalWhile(node, scope)
	default:
		return normal(Null())
	}
}

func normal(value Value) (evalResult, error) {
	return evalResult{value: value, signal: noSignal}, nil
}

func literalValue(node ir.Node) Value {
	switch node.DataType {
	case "boolean":
		return Boolean(node.Value == "true")
	case "integer":
		value, err := strconv.ParseInt(node.Value, 10, 64)
		if err != nil {
			return Integer(0)
		}
		return Integer(value)
	case "float":
		value, err := strconv.ParseFloat(node.Value, 64)
		if err != nil {
			return Float(0)
		}
		return Float(value)
	case "string":
		return String(node.Value)
	default:
		return String(node.Value)
	}
}

func (i *Interpreter) evalSequence(nodes []ir.Node, scope *Scope) (evalResult, error) {
	last := Null()

	for _, child := range nodes {
		result, err := i.eval(child, scope)
		if err != nil {
			return result, err
		}

		last = result.value
		if result.signal != noSignal {
			return result, nil
		}
	}

	return normal(last)
}

func (i *Interpreter) evalIdentifier(node ir.Node, scope *Scope) (evalResult, error) {
	if value, ok := scope.Get(node.Name); ok {
		return normal(value)
	}

	return normal(Null())
}

func (i *Interpreter) evalAssign(node ir.Node, scope *Scope) (evalResult, error) {
	if len(node.Children) < 2 {
		return normal(Null())
	}

	target := node.Children[0]
	valueNode := node.Children[len(node.Children)-1]
	value, err := i.evalAssignmentValue(valueNode, scope)
	if err != nil {
		return value, err
	}
	if value.signal != noSignal {
		return value, nil
	}

	switch target.Type {
	case ir.Identifier:
		scope.Set(target.Name, value.value)
	case ir.Member:
		if err := i.assignMember(target, value.value, scope); err != nil {
			return value, err
		}
	default:
		return normal(value.value)
	}

	return normal(value.value)
}

func (i *Interpreter) evalAssignmentValue(node ir.Node, scope *Scope) (evalResult, error) {
	if node.Type == ir.Block && len(node.Children) == 0 {
		return normal(Object(map[string]Value{}))
	}

	return i.eval(node, scope)
}

func (i *Interpreter) evalImport(node ir.Node, scope *Scope) (evalResult, error) {
	module, err := i.loadModule(node.Name)
	if err != nil {
		return evalResult{}, err
	}

	if node.DataType == "from" {
		for _, imported := range node.Children {
			value, ok := module.Object[imported.Name]
			if !ok {
				return evalResult{}, runtimeError(node, "module %q has no export %q", node.Name, imported.Name)
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

func (i *Interpreter) bindImportedModule(scope *Scope, parts []string, module Value) {
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

func (i *Interpreter) loadModule(path string) (Value, error) {
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

	data, err := os.ReadFile(modulePath)
	if err != nil {
		return Null(), err
	}

	var l lexer.Lexer
	l.Set(string(data))
	l.Tokenize()

	var p parser.Parser
	p.Set(l.Get())
	p.Parse()

	program := ir.Lower(p.Get())
	moduleScope := NewScope(nil)

	previousFile := i.currentFile
	i.currentFile = modulePath
	result, err := i.eval(program, moduleScope)
	i.currentFile = previousFile
	if err != nil {
		return result.value, err
	}

	for name, value := range moduleScope.Snapshot() {
		if strings.HasPrefix(name, "_") {
			continue
		}
		module.Object[name] = value
	}

	entry.state = moduleLoaded
	entry.value = module
	return module, nil
}

func (i *Interpreter) resolveModulePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("[blbx][runtime] empty import path")
	}

	parts := strings.Split(path, ".")
	for _, part := range parts {
		if part == "" {
			return "", fmt.Errorf("[blbx][runtime] invalid import path %q", path)
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
			return "", fmt.Errorf("[blbx][runtime] package %q requires %s", strings.Join(parts[:index], "."), initPath)
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

	return "", fmt.Errorf("[blbx][runtime] could not resolve import %q from %s", path, root)
}

func (i *Interpreter) assignMember(target ir.Node, value Value, scope *Scope) error {
	if len(target.Children) < 2 {
		return nil
	}

	objectRef := target.Children[0]
	propertyRef := target.Children[1]

	if objectRef.Type != ir.Identifier || propertyRef.Type != ir.Identifier {
		return nil
	}

	object, ok := scope.Get(objectRef.Name)
	if !ok || object.Kind != ObjectKind {
		object = Object(map[string]Value{})
	}

	object.Object[propertyRef.Name] = value
	scope.Set(objectRef.Name, object)
	return nil
}

func (i *Interpreter) evalBlock(node ir.Node, scope *Scope) (evalResult, error) {
	if isObjectLiteral(node) {
		return i.evalObjectLiteral(node, scope)
	}

	return i.evalSequence(node.Children, scope)
}

func isObjectLiteral(node ir.Node) bool {
	if len(node.Children) == 0 {
		return false
	}

	for _, child := range node.Children {
		if child.Type != ir.Assign || len(child.Children) < 2 {
			return false
		}
		if child.Children[0].Type != ir.Literal || child.Children[0].DataType != "string" {
			return false
		}
	}

	return true
}

func (i *Interpreter) evalObjectLiteral(node ir.Node, scope *Scope) (evalResult, error) {
	object := map[string]Value{}

	for _, child := range node.Children {
		key := child.Children[0].Value
		value, err := i.eval(child.Children[len(child.Children)-1], scope)
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

func (i *Interpreter) evalArray(node ir.Node, scope *Scope) (evalResult, error) {
	values := []Value{}

	for _, child := range node.Children {
		value, err := i.eval(child, scope)
		if err != nil {
			return value, err
		}
		if value.signal != noSignal {
			return value, nil
		}
		values = append(values, value.value)
	}

	return normal(Array(values))
}

func (i *Interpreter) evalTuple(node ir.Node, scope *Scope) (evalResult, error) {
	return i.evalArray(node, scope)
}

func (i *Interpreter) evalFunction(node ir.Node, scope *Scope) (evalResult, error) {
	fn := &Function{
		Name:   node.Name,
		Params: []Param{},
		Env:    scope,
	}

	for _, child := range node.Children {
		if child.Type == ir.Identifier {
			fn.Params = append(fn.Params, Param{
				Name:    child.Name,
				Default: Null(),
			})
			continue
		}

		if child.Type == ir.Tuple {
			for _, tupleChild := range child.Children {
				if tupleChild.Type == ir.Identifier {
					fn.Params = append(fn.Params, Param{
						Name:    tupleChild.Name,
						Default: Null(),
					})
				}
			}
			continue
		}

		if child.Type == ir.Assign && len(child.Children) >= 2 && child.Children[0].Type == ir.Identifier {
			defaultValue, err := i.eval(child.Children[1], scope)
			if err != nil {
				return defaultValue, err
			}
			fn.Params = append(fn.Params, Param{
				Name:    child.Children[0].Name,
				Default: defaultValue.value,
			})
			continue
		}

		if child.Type == ir.Block {
			fn.Body = child
			continue
		}
	}

	return normal(FunctionValue(fn))
}

func (i *Interpreter) evalReturn(node ir.Node, scope *Scope) (evalResult, error) {
	if len(node.Children) == 0 {
		return evalResult{value: Null(), signal: returnSignal}, nil
	}

	value, err := i.eval(node.Children[0], scope)
	if err != nil {
		return value, err
	}
	value.signal = returnSignal
	return value, nil
}

func (i *Interpreter) evalAssert(node ir.Node, scope *Scope) (evalResult, error) {
	if len(node.Children) == 0 {
		return normal(Null())
	}

	value, err := i.eval(node.Children[0], scope)
	if err != nil {
		return value, err
	}
	if value.signal != noSignal {
		return value, nil
	}

	if value.value.IsTruthy() {
		return evalResult{value: value.value, signal: assertSignal}, nil
	}

	return normal(Null())
}

func (i *Interpreter) evalCall(node ir.Node, scope *Scope) (evalResult, error) {
	if node.Name == "print" {
		return i.evalPrint(node, scope)
	}

	if node.Name == "input" {
		return i.evalInput(node, scope)
	}

	if len(node.Children) > 0 && node.Children[0].Type == ir.Member {
		return i.evalMethodCall(node, scope)
	}

	if len(node.Children) > 0 && node.Children[0].Type == ir.Index {
		return i.evalExpressionCall(node, scope)
	}

	callee, ok := scope.Get(node.Name)
	if !ok || callee.Kind != FunctionKind {
		return normal(Null())
	}

	args, result, err := i.evalArgs(node.Children, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	return i.callFunction(callee.Function, args)
}

func (i *Interpreter) evalExpressionCall(node ir.Node, scope *Scope) (evalResult, error) {
	callee, err := i.eval(node.Children[0], scope)
	if err != nil {
		return callee, err
	}
	if callee.signal != noSignal {
		return callee, nil
	}

	if callee.value.Kind != FunctionKind {
		return normal(Null())
	}

	args, result, err := i.evalArgs(node.Children[1:], scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	return i.callFunction(callee.value.Function, args)
}

func (i *Interpreter) evalPrint(node ir.Node, scope *Scope) (evalResult, error) {
	args, result, err := i.evalArgs(node.Children, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	for _, arg := range args {
		text := arg.Display()
		i.Output = append(i.Output, text)
		fmt.Println(text)
	}

	return normal(Null())
}

func (i *Interpreter) evalInput(node ir.Node, scope *Scope) (evalResult, error) {
	args, result, err := i.evalArgs(node.Children, scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	if len(args) > 0 {
		fmt.Print(args[0].Display())
	}

	text, err := i.Input.ReadString('\n')
	if err != nil && err != io.EOF {
		return evalResult{}, runtimeError(node, "input failed: %v", err)
	}

	text = strings.TrimSuffix(text, "\n")
	text = strings.TrimSuffix(text, "\r")

	return normal(String(text))
}

func (i *Interpreter) evalMethodCall(node ir.Node, scope *Scope) (evalResult, error) {
	member := node.Children[0]
	if len(member.Children) < 2 {
		return normal(Null())
	}

	receiver, err := i.eval(member.Children[0], scope)
	if err != nil {
		return receiver, err
	}
	if receiver.signal != noSignal {
		return receiver, nil
	}

	method := member.Children[1].Name
	args, result, err := i.evalArgs(node.Children[1:], scope)
	if err != nil || result.signal != noSignal {
		return result, err
	}

	if method == "continue" {
		return evalResult{value: Null(), signal: continueSignal}, nil
	}

	if method == "break" {
		return evalResult{value: Null(), signal: breakSignal}, nil
	}

	if receiver.value.Kind == ObjectKind {
		if property, ok := receiver.value.Object[method]; ok && property.Kind == FunctionKind {
			return i.callFunction(property.Function, args)
		}
	}

	return normal(i.callMethod(receiver.value, method, args))
}

func (i *Interpreter) evalArgs(nodes []ir.Node, scope *Scope) ([]Value, evalResult, error) {
	args := []Value{}

	for _, child := range nodes {
		value, err := i.eval(child, scope)
		if err != nil {
			return args, value, err
		}
		if value.signal != noSignal {
			return args, value, nil
		}
		args = append(args, value.value)
	}

	return args, evalResult{}, nil
}

func (i *Interpreter) callFunction(fn *Function, args []Value) (evalResult, error) {
	if fn == nil {
		return normal(Null())
	}

	callScope := NewScope(fn.Env)
	for index, param := range fn.Params {
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

	if result.signal == returnSignal || result.signal == assertSignal {
		return normal(result.value)
	}

	return result, nil
}

func (i *Interpreter) callMethod(receiver Value, method string, args []Value) Value {
	switch method {
	case "concat":
		if receiver.Kind == StringKind && len(args) > 0 {
			return String(receiver.String + args[0].Display())
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
		return Boolean(!receiver.IsTruthy())
	case "type":
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

func (i *Interpreter) evalMember(node ir.Node, scope *Scope) (evalResult, error) {
	if len(node.Children) < 2 {
		return normal(Null())
	}

	receiver, err := i.eval(node.Children[0], scope)
	if err != nil {
		return receiver, err
	}
	if receiver.signal != noSignal {
		return receiver, nil
	}

	property := node.Children[1].Name
	if property == "length" {
		return normal(lengthOf(receiver.value))
	}

	if receiver.value.Kind == ObjectKind {
		if value, ok := receiver.value.Object[property]; ok {
			return normal(value)
		}
	}

	return normal(Null())
}

func (i *Interpreter) evalIndex(node ir.Node, scope *Scope) (evalResult, error) {
	if len(node.Children) < 2 {
		return normal(Null())
	}

	target, err := i.eval(node.Children[0], scope)
	if err != nil {
		return target, err
	}
	index, err := i.eval(node.Children[1], scope)
	if err != nil {
		return index, err
	}

	if target.value.Kind == ObjectKind && index.value.Kind == StringKind {
		if value, ok := target.value.Object[index.value.String]; ok {
			return normal(value)
		}

		return normal(Null())
	}

	if target.value.Kind == ArrayKind && index.value.Kind == IntegerKind {
		arrayIndex := int(index.value.Integer)
		if arrayIndex < 0 || arrayIndex >= len(target.value.Array) {
			return normal(Null())
		}

		return normal(target.value.Array[arrayIndex])
	}

	return normal(Null())
}

func (i *Interpreter) evalBinary(node ir.Node, scope *Scope) (evalResult, error) {
	if len(node.Children) < 2 {
		return normal(Null())
	}

	left, err := i.eval(node.Children[0], scope)
	if err != nil {
		return left, err
	}
	right, err := i.eval(node.Children[1], scope)
	if err != nil {
		return right, err
	}

	switch node.Name {
	case "*":
		if left.value.Kind == FloatKind || right.value.Kind == FloatKind {
			return normal(Float(left.value.number() * right.value.number()))
		}
		return normal(Integer(left.value.Integer * right.value.Integer))
	default:
		return normal(Null())
	}
}

func (i *Interpreter) evalIf(node ir.Node, scope *Scope) (evalResult, error) {
	if len(node.Children) == 0 {
		return normal(Null())
	}

	condition, err := i.eval(node.Children[0], scope)
	if err != nil {
		return condition, err
	}
	if condition.signal != noSignal {
		return condition, nil
	}

	branchIndex := 2
	if condition.value.IsTruthy() {
		branchIndex = 1
	}
	if branchIndex >= len(node.Children) {
		return normal(Null())
	}

	result, err := i.evalScopedValue(node.Children[branchIndex], scope)
	if result.signal == assertSignal {
		return normal(result.value)
	}
	return result, err
}

func (i *Interpreter) evalFor(node ir.Node, scope *Scope) (evalResult, error) {
	if len(node.Children) < 2 {
		return normal(Null())
	}

	iterable, err := i.eval(node.Children[0], scope)
	if err != nil {
		return iterable, err
	}
	callback, err := i.eval(node.Children[1], scope)
	if err != nil {
		return callback, err
	}

	if iterable.value.Kind != ArrayKind || callback.value.Kind != FunctionKind {
		return normal(Null())
	}

	last := Null()
	step := int64(1)
	for index, elem := range iterable.value.Array {
		loopValue := Object(map[string]Value{
			"elem": elem,
			"idx":  Integer(int64(index)),
			"step": Integer(step),
		})
		result, err := i.callFunction(callback.value.Function, []Value{loopValue})
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
		if result.signal == returnSignal || result.signal == assertSignal {
			return result, nil
		}
		step++
	}

	return normal(last)
}

func (i *Interpreter) evalWhile(node ir.Node, scope *Scope) (evalResult, error) {
	if len(node.Children) < 2 {
		return normal(Null())
	}

	last := Null()
	for {
		condition, err := i.eval(node.Children[0], scope)
		if err != nil {
			return condition, err
		}
		if !condition.value.IsTruthy() {
			break
		}

		result, err := i.evalScopedValue(node.Children[1], scope)
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
		if result.signal == returnSignal || result.signal == assertSignal {
			return result, nil
		}
	}

	return normal(last)
}

func (i *Interpreter) evalScopedValue(node ir.Node, scope *Scope) (evalResult, error) {
	result, err := i.eval(node, scope)
	if err != nil {
		return result, err
	}

	if result.value.Kind == FunctionKind {
		return i.callFunction(result.value.Function, []Value{})
	}

	if result.signal == assertSignal {
		return normal(result.value)
	}

	return result, nil
}

func runtimeError(node ir.Node, format string, args ...interface{}) error {
	return fmt.Errorf("[blbx][runtime] line %d: %s", node.Line, fmt.Sprintf(format, args...))
}

func lengthOf(value Value) Value {
	switch value.Kind {
	case ArrayKind:
		return Integer(int64(len(value.Array)))
	case StringKind:
		return Integer(int64(len(value.String)))
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
