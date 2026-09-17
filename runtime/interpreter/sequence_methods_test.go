package interpreter

import (
	"blbx_lang/syntax/check"
	"blbx_lang/syntax/ir"
	"io"
	"testing"
)

func TestSequenceMethods(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{`("").concat("1", "2", 3)`, "123"},
		{`("  a\t b\n").split().join("-")`, "a-b"},
		{`("a,,b,").split(",").length()`, "4"},
		{`("é😀").split("").join("|")`, "é|😀"},
		{`("-").join([1, true, null])`, "1-true-null"},
		{`(["a", "b"]).join()`, "ab"},
		{`("é😀").length`, "2"},
		{`("é😀x").slice(1, 2)`, "😀"},
		{`("é😀x").char_at(1)`, "😀"},
		{`("é😀x").char_at(("-1").to_int())`, "x"},
		{`("é😀x").index_of("x")`, "2"},
		{`("abc").index_of("z")`, "-1"},
		{`("abc").contains("b")`, "true"},
		{`("abc").starts_with("ab")`, "true"},
		{`("abc").ends_with("bc")`, "true"},
		{`("éa").upper().lower()`, "éa"},
		{`("  a  ").trim()`, "a"},
		{`("  a  ").trim_start()`, "a  "},
		{`("  a  ").trim_end()`, "  a"},
		{`("aba").replace("a", "x")`, "xba"},
		{`("aba").replace_all("a", "x")`, "xbx"},
		{`("é😀").reverse()`, "😀é"},
		{`("ab").repeat(3)`, "ababab"},
		{`("").is_empty()`, "true"},
		{`([1, 2]).concat([3], [4]).append(5).reverse().slice(1, 4).join(",")`, "4,3,2"},
		{`([1, 2]).contains(2)`, "true"},
		{`([1, 2]).index_of(2)`, "1"},
		{`([1, 2]).first()`, "1"},
		{`([1, 2]).last()`, "2"},
		{`([]).is_empty()`, "true"},
		{`([]).join(",")`, ""},
		{`([]).first()`, "null"},
		{`([1, 2, 3]).slice(("-2").to_int()).join()`, "23"},
		{`([1, 2, 3]).slice(2, 1).length()`, "0"},
		{`("abc").slice(99)`, ""},
		{`("abc").char_at(99)`, "null"},
		{`("a").repeat(9223372036854775807)`, "null"},
		{`("abc").split(1)`, "null"},
		{`([1]).join(2)`, "null"},
		{`([1]).concat(2)`, "null"},
		{`([1]).split()`, "null"},
		{`("a").slice("bad")`, "null"},
		{`("a").upper(1)`, "null"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			nodes, diagnostics := check.Source("sequence.bx", tc.source)
			if len(diagnostics) != 0 {
				t.Fatal(diagnostics)
			}
			runtime := New()
			runtime.SetOutput(io.Discard)
			result, err := runtime.Execute(ir.Lower(nodes))
			if err != nil {
				t.Fatal(err)
			}
			if got := result.Last.Display(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestArrayMethodsDoNotMutateReceiver(t *testing.T) {
	backing := []Value{Integer(1), Integer(2), Integer(3), Integer(99)}
	original := Array(backing[:3])
	for _, tc := range []struct {
		name string
		args []Value
	}{
		{"reverse", nil}, {"slice", nil}, {"concat", []Value{Array([]Value{Integer(4)})}}, {"append", []Value{Integer(4)}},
	} {
		result, handled := sequenceMethod(original, tc.name, tc.args)
		if !handled || result.Kind != ArrayKind {
			t.Fatal(tc.name, result)
		}
		result.Array[0] = Integer(77)
		if backing[0].Integer != 1 || backing[3].Integer != 99 {
			t.Fatalf("%s mutated backing array", tc.name)
		}
	}
}
