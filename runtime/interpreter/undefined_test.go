package interpreter

import (
	"strings"
	"testing"
)

func TestUndefinedReferences(t *testing.T) {
	for _, tc := range []struct{ source, message string }{
		{`print(missing)`, `undefined name "missing"`},
		{`missing()`, `undefined function "missing"`},
		{`print(missing.member)`, `undefined name "missing"`},
		{`obj = {} print(obj.missing)`, `undefined member "missing"`},
		{`obj = {} obj.missing()`, `undefined method "missing"`},
		{`obj = {} print(obj["missing"])`, `undefined member "missing"`},
		{`x = 1 x()`, `not callable`},
		{`obj = {"x": null} obj.x()`, `not callable`},
		{`obj = {"x": null} obj["x"]()`, `not callable`},
		{`unknown.x = 1`, `undefined name "unknown"`},
		{`obj = 1 obj.x = 2`, `cannot assign member`},
		{`("hello").typo()`, `undefined method "typo"`},
		{`class A {} a = A() print(a.indexes)`, `undefined member "indexes"`},
		{`import std.math as math print(math.missing)`, `undefined member "missing"`},
	} {
		_, err := runClassSource(t, tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.message) {
			t.Errorf("%s: got %v, want %s", tc.source, err, tc.message)
		}
	}
}

func TestNullAndDeclarationsRemainValid(t *testing.T) {
	result, err := runClassSource(t, `
x = null
obj = {}
obj.x = null
f = () => { return null }
print(x)
print(obj.x)
print(obj["x"])
print(f())
print(("bad").to_int())
print([1][9])
`)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(result.Output, []string{"null", "null", "null", "null", "null", "null"}) {
		t.Fatal(result.Output)
	}
}
