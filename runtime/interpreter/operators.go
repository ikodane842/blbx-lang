package interpreter

// Method syntax uses the same hooks as infix multiplication. Explicit ordinary
// methods win over hooks, and hooks are resolved only on object receivers.
var operatorHooks = map[string]string{
	"add": "__add__", "sub": "__sub__", "mul": "__mul__", "div": "__div__", "mod": "__mod__",
	"eq": "__eq__", "neq": "__ne__", "lt": "__lt__", "lte": "__le__", "gt": "__gt__", "gte": "__ge__",
	"and": "__and__", "or": "__or__", "not": "__not__",
}

func operatorMethod(receiver Value, name string) (Value, bool) {
	if receiver.Kind != ObjectKind {
		return Null(), false
	}
	if method, ok := receiver.Object[name]; ok {
		return method, true
	}
	if hook, ok := operatorHooks[name]; ok {
		method, exists := receiver.Object[hook]
		return method, exists
	}
	return Null(), false
}
