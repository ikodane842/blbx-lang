package interpreter

import (
	"fmt"
	"testing"
)

func TestBooleanOperators(t *testing.T) {
	for _, left := range []bool{false, true} {
		for _, right := range []bool{false, true} {
			result, err := runClassSource(t, fmt.Sprintf("print((%t).and(%t)) print((%t).or(%t)) print((%t).not())", left, right, left, right, left))
			if err != nil {
				t.Fatal(err)
			}
			want := []string{fmt.Sprint(left && right), fmt.Sprint(left || right), fmt.Sprint(!left)}
			if !sameStrings(result.Output, want) {
				t.Fatal(result.Output, want)
			}
		}
	}
}

func TestLogicalShortCircuit(t *testing.T) {
	result, err := runClassSource(t, `
calls = 0
touch = () => { calls = calls.add(1) return true }
print((false).and(missing()))
print((true).or(missing()))
print((false).and(touch()))
print((true).or(touch()))
print(calls)
print((true).and(touch()))
print((false).or(touch()))
print(calls)
print((2).lt(3).and((4).gt(1)).not())
obj = {}
obj.and = (value) => { return value }
print(obj.and("custom"))
`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"false", "true", "false", "true", "0", "true", "true", "2", "false", "custom"}
	if !sameStrings(result.Output, want) {
		t.Fatal(result.Output)
	}
}

func TestLogicalArgumentErrors(t *testing.T) {
	for _, source := range []string{`true.and()`, `false.or()`, `true.not(1)`, `false.and(true, false)`, `true.or(true, false)`, `true.and(1)`, `false.or(null)`, `true.and(missing)`} {
		if _, err := runClassSource(t, source); err == nil {
			t.Fatal("expected error", source)
		}
	}
}
