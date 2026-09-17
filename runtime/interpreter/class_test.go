package interpreter

import (
	"blbx_lang/syntax/check"
	"blbx_lang/syntax/ir"
	"io"
	"strings"
	"testing"
)

func runClassSource(t *testing.T, source string) (Result, error) {
	t.Helper()
	nodes, diagnostics := check.Source("class.bx", source)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	runtime := New()
	runtime.SetOutput(io.Discard)
	return runtime.Execute(ir.Lower(nodes))
}

func TestClassConstructorPlacement(t *testing.T) {
	constructor := `(_type, _value) => { self.type = _type self.value = _value self.set() }`
	members := []string{`get_value = () => { return self.value }`, `set = () => { self.ready = true }`, `fallback = defaultValue`}
	for position := 0; position <= len(members); position++ {
		body := append([]string{}, members[:position]...)
		body = append(body, constructor)
		body = append(body, members[position:]...)
		source := "defaultValue = 0\nclass Token {\n" + strings.Join(body, "\n") + `
}
a = Token("STRING", "first")
b = Token("INTEGER", "second")
print(a)
print(a.get_value())
print(b.get_value())
print(a.type)
print(a.ready)
print(typeof(Token))
print(typeof(a))
saved = a.get_value
print(saved())
print(a["get_value"]())
`
		result, err := runClassSource(t, source)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"<Token instance>", "first", "second", "STRING", "true", "class", "object", "first", "first"}
		if !sameStrings(result.Output, want) {
			t.Fatalf("position %d: %v", position, result.Output)
		}
	}
}

func TestPlainObjectSelf(t *testing.T) {
	result, err := runClassSource(t, `
a = {}
a.value = "a"
a.get = () => { return self.value }
b = {}
b.value = "b"
b.get = a.get
print(a.get())
print(b.get())
saved = a.get
print(saved())
print(b["get"]())
c = { "value": "c", "get": () => { return self.value } }
print(c.get())
a.nested = () => { return () => { return self.value } }
callback = a.nested()
print(callback())
`)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(result.Output, []string{"a", "b", "a", "b", "c", "a"}) {
		t.Fatal(result.Output)
	}
}

func TestClassesWithoutConstructorAndFreshDefaults(t *testing.T) {
	result, err := runClassSource(t, `
class Box { data = {} set = (v) => { self.value = v } }
a = Box()
b = Box()
a.set(1)
b.set(2)
print(a.value)
print(b.value)
class Returner { () => { return 99 } }
print(Returner())
`)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(result.Output, []string{"1", "2", "<Returner instance>"}) {
		t.Fatal(result.Output)
	}
	a, b := result.Globals["a"], result.Globals["b"]
	a.Object["data"].Object["changed"] = Boolean(true)
	if _, shared := b.Object["data"].Object["changed"]; shared {
		t.Fatal("instance defaults are shared")
	}
}

func TestClassAndSelfErrors(t *testing.T) {
	for _, source := range []string{
		`class {}`, `class A { () => {} () => {} }`, `class A { print(1) }`, `self = 1`, `f = (self) => {}`,
	} {
		_, diagnostics := check.Source("class.bx", source)
		if len(diagnostics) == 0 {
			t.Fatalf("expected diagnostics for %s", source)
		}
	}
	for _, source := range []string{`print(self)`, `self.value = 1`, `class A {} A(1)`} {
		_, err := runClassSource(t, source)
		if err == nil {
			t.Fatalf("expected runtime error for %s", source)
		}
	}
}

func TestClassClosuresAndNestedSelf(t *testing.T) {
	result, err := runClassSource(t, `
class Counter {
    data = {}
    () => { self.data.value = 0 }
    next = () => { self.data.value = self.data.value.add(1) return self.data.value }
    callback = () => { return () => { return self.next() } }
}
a = Counter()
b = Counter()
fn = a.callback()
print(fn())
print(fn())
print(b.next())
`)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(result.Output, []string{"1", "2", "1"}) {
		t.Fatal(result.Output)
	}
}
