package interpreter

import (
	"blbx_lang/syntax/check"
	"blbx_lang/syntax/ir"
	"io"
	"testing"
)

func TestDestructuring(t *testing.T) {
	result, err := runClassSource(t, `
array = [1, 2, [3, 4], 5]
[a, b, [c, d], ...tail] = array
print([a, b, c, d].join(","))
print(tail.join(","))
[a, b] = [b, a]
print([a, b].join(","))
person = { "name": "Ada", "address": { "city": "London" }, "age": 36 }
{name: label, address: {city}, ...other} = person
print(label)
print(city)
print(other.age)
{age} = other
print(age)
[value] = [null]
print(value)
{ "name": renamed } = person
print(renamed)
[] = []
{} = person
[...all] = []
print(all.length)
`)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(result.Output, []string{"1,2,3,4", "5", "2,1", "Ada", "London", "36", "36", "null", "Ada", "0"}) {
		t.Fatal(result.Output)
	}
}

func TestDestructuringRHSOnceAndBoundMethods(t *testing.T) {
	result, err := runClassSource(t, `
count = 0
make = () => { count = count.add(1) return [1, 2] }
[first, second] = make()
print(count)
class Item { () => { self.value = "ok" } get = () => { return self.value } }
item = Item()
{get, value} = item
print(get())
print(value)
`)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(result.Output, []string{"1", "ok", "ok"}) {
		t.Fatal(result.Output)
	}
}

func TestDestructuringErrorsAreAtomic(t *testing.T) {
	for _, source := range []string{
		"a = 99\n[a, b] = [1]", `a = 99 {x: a, missing} = {"x": 1}`,
		"a = 99\n[a, [b]] = [1, 2]", `a = 99 {x: a} = []`,
	} {
		nodes, ds := check.Source("pattern.bx", source)
		if len(ds) != 0 {
			t.Fatal(ds)
		}
		runtime := New()
		runtime.SetOutput(io.Discard)
		if _, err := runtime.Execute(ir.Lower(nodes)); err == nil {
			t.Fatal("expected error", source)
		}
		value, _ := runtime.global.Get("a")
		if value.Integer != 99 {
			t.Fatal("partial write", source)
		}
	}
}

func TestInvalidDestructuringPatterns(t *testing.T) {
	for _, source := range []string{
		`[a, a] = [1, 2]`, `[self] = [1]`, `[true] = [1]`, `[1] = [1]`,
		`[...rest, last] = [1, 2]`, `{a, a} = obj`, `{a: x, b: x} = obj`,
		`{...rest, a} = obj`, `[a.b] = [1]`,
	} {
		_, diagnostics := check.Source("pattern.bx", source)
		if len(diagnostics) == 0 {
			t.Fatal("accepted invalid pattern", source)
		}
	}
}
