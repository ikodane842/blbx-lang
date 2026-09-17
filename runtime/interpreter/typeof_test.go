package interpreter

import (
	"blbx_lang/syntax/check"
	"blbx_lang/syntax/ir"
	"io"
	"strings"
	"testing"
)

func TestTypeofAllTypes(t *testing.T) {
	for _, tc := range []struct{ expression, kind string }{
		{"null", "null"}, {"true", "boolean"}, {"42", "integer"},
		{"1.5", "float"}, {`"hello"`, "string"}, {"[1, 2]", "array"},
		{"{}", "object"}, {"fn", "function"}, {"[1][9]", "null"},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			source := "fn = () => {}\nx = " + tc.expression + "\nprint(typeof(x)) print(x.typeof) print((x).typeof) print(typeof(x.typeof)) print(x.typeof())"
			nodes, diagnostics := check.Source("test.bx", source)
			if len(diagnostics) != 0 {
				t.Fatal(diagnostics)
			}
			runtime := New()
			runtime.SetOutput(io.Discard)
			result, err := runtime.Execute(ir.Lower(nodes))
			if err != nil {
				t.Fatal(err)
			}
			want := []string{tc.kind, tc.kind, tc.kind, "string", tc.kind}
			if !sameStrings(result.Output, want) {
				t.Fatalf("got %v, want %v", result.Output, want)
			}
		})
	}
}

func TestTypeofArity(t *testing.T) {
	for _, source := range []string{"typeof()", "typeof(1, 2)"} {
		nodes, diagnostics := check.Source("test.bx", source)
		if len(diagnostics) != 0 {
			t.Fatal(diagnostics)
		}
		_, err := New().Execute(ir.Lower(nodes))
		if err == nil || !strings.Contains(err.Error(), "exactly one argument") {
			t.Fatalf("got %v", err)
		}
	}
}

func TestRecursiveArrayTraversalWithTypeofMethod(t *testing.T) {
	source := `
array = [1, 2, 3, 4, [5, 6, 7], [8, 9]]
recurse = (_array) => {
    for(
        _array,
        (__) => {
            if(
                (__.elem().typeof().eq("array")),
                () => { recurse(__.elem()) },
                () => { print(__.elem()) }
            )
        }
    )
}
recurse(array)
`
	nodes, diagnostics := check.Source("recursive.bx", source)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	runtime := New()
	runtime.SetOutput(io.Discard)
	result, err := runtime.Execute(ir.Lower(nodes))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("got %v, want %v", result.Output, want)
	}
}

func TestTypeofEvaluatesOnceAndPreservesObjectMethods(t *testing.T) {
	source := `
count = 0
next = () => { count = count.add(1) return null }
print(typeof(next()))
print(count)
obj = {}
obj.typeof = () => { return "custom" }
print(obj.typeof())
print(obj.typeof)
`
	nodes, diagnostics := check.Source("test.bx", source)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	runtime := New()
	runtime.SetOutput(io.Discard)
	result, err := runtime.Execute(ir.Lower(nodes))
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(result.Output, []string{"null", "1", "custom", "object"}) {
		t.Fatal(result.Output)
	}
}
