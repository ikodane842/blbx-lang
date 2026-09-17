package interpreter

import (
	"blbx_lang/syntax/check"
	"io"
	"path/filepath"
	"testing"
)

func TestInheritance(t *testing.T) {
	result, err := runClassSource(t, `
class Base {
    data = {}
    kind = "base"
    (name) => { self.name = name self.result = self.describe() }
    describe = () => { return "base" }
    get_name = () => { return self.name }
}
class Child extends Base {
    kind = "child"
    describe = () => { return self.kind }
}
class Grandchild extends Child {}
a = Grandchild("one")
b = Grandchild("two")
print(a)
print(a.get_name())
print(b.get_name())
print(a.result)
print(a.kind)
saved = a.get_name
print(saved())
class Custom extends Base { () => { self.name = "custom" } }
print(Custom().get_name())
`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"<Grandchild instance>", "one", "two", "child", "child", "one", "custom"}
	if !sameStrings(result.Output, want) {
		t.Fatal(result.Output)
	}
	a, b := result.Globals["a"], result.Globals["b"]
	a.Object["data"].Object["changed"] = Boolean(true)
	if _, shared := b.Object["data"].Object["changed"]; shared {
		t.Fatal("inherited defaults shared")
	}
}

func TestObjectIndexesAndValues(t *testing.T) {
	result, err := runClassSource(t, `
o = { "z": 2, "a": 1 }
print(o.indexes.join(","))
print(o.values.join(","))
print(o.indexes().join(","))
print(o.values().join(","))
empty = {}
print(empty.indexes.length)
class A { values = "custom" }
a = A()
print(a.values)
print(a.indexes())
`)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(result.Output, []string{"a,z", "1,2", "a,z", "1,2", "0", "custom", "null"}) {
		t.Fatal(result.Output)
	}
}

func TestInheritanceErrors(t *testing.T) {
	for _, source := range []string{`class A extends {}`, `class A extends B. {}`, `class A extends B, C {}`} {
		_, diagnostics := check.Source("bad.bx", source)
		if len(diagnostics) == 0 {
			t.Fatal(source)
		}
	}
	for _, source := range []string{`class A extends Missing {}`, `base = {} class A extends base {}`, `class A extends A {}`} {
		if _, err := runClassSource(t, source); err == nil {
			t.Fatal(source)
		}
	}
}

func TestImportedBaseKeepsLexicalScope(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "base.bx"), `label = "base" class Base { get = () => { return label } }`)
	main := filepath.Join(root, "main.bx")
	writeTestFile(t, main, `import base label = "child" class Child extends base.Base {} print(Child().get())`)
	runtime := New()
	runtime.SetOutput(io.Discard)
	result, err := runtime.ExecuteFile(main)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(result.Output, []string{"base"}) {
		t.Fatal(result.Output)
	}
}
