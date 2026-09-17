package interpreter

import (
	"blbx_lang/syntax/check"
	"blbx_lang/syntax/ir"
	"io"
	"testing"
)

func TestSingletonConversions(t *testing.T) {
	source := `
print((11).to_str())
print((11).to_str().typeof())
print(("1").to_int().typeof())
print(("1").to_float().typeof())
print(("12").to_int().add(3))
print(("1.5").to_float().mul(2))
print((true).not())
print((false).not())
print((null).to_str())
print((3.9).to_int())
print((2).to_float().typeof())
print(("invalid").to_int())
print(("NaN").to_float())
print(("9223372036854775808").to_int())
print((true).to_int())
print((1).not())
print(("1").to_int(2))
`
	nodes, diagnostics := check.Source("singleton.bx", source)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	runtime := New()
	runtime.SetOutput(io.Discard)
	result, err := runtime.Execute(ir.Lower(nodes))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"11", "string", "integer", "float", "15", "3", "false", "true", "null", "3", "float", "null", "null", "null", "null", "null", "null"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("got %v, want %v", result.Output, want)
	}
}
