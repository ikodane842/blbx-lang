package interpreter

import (
	"blbx_lang/syntax/ir"
	"blbx_lang/syntax/lexer"
	"blbx_lang/syntax/parser"
	"strings"
	"testing"
)

func TestArrayLengthSingletonCalls(t *testing.T) {
	source := `
array = [1, 2, 3]
print((array).length)
print(array.length())
print((array).length())
`

	var l lexer.Lexer
	l.Set(source)
	l.Tokenize()

	var p parser.Parser
	p.Set(l.Get())
	p.Parse()

	runtime := New()
	result, err := runtime.Execute(ir.Lower(p.Get()))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	want := []string{"3", "3", "3"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("output = %#v, want %#v", result.Output, want)
	}
}

func TestWhileWithSingletonMathAndComparisonMethods(t *testing.T) {
	source := `
list = [
    "stand",
    "up",
    "and",
    "fight"
]

limit = (list).length
start = 0

while(
    start.lt(limit),
    () => {
        print(list[start])
        start = start.add(1)
    }
)
`

	var l lexer.Lexer
	l.Set(source)
	l.Tokenize()

	var p parser.Parser
	p.Set(l.Get())
	p.Parse()

	runtime := New()
	result, err := runtime.Execute(ir.Lower(p.Get()))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	want := []string{"stand", "up", "and", "fight"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("output = %#v, want %#v", result.Output, want)
	}
}

func TestSingletonComparisonAndMathMethods(t *testing.T) {
	source := `
print((3).gt(2))
print((3).gte(3))
print((2).lt(3))
print((2).lte(2))
print((2).eq(2))
print((2).neq(3))
print(false.not())
print((2).add(3))
print((5).sub(2))
print((3).mul(4))
print((8).div(2))
print((8).mod(3))
`

	var l lexer.Lexer
	l.Set(source)
	l.Tokenize()

	var p parser.Parser
	p.Set(l.Get())
	p.Parse()

	runtime := New()
	result, err := runtime.Execute(ir.Lower(p.Get()))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	want := []string{"true", "true", "true", "true", "true", "true", "true", "5", "3", "12", "4", "2"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("output = %#v, want %#v", result.Output, want)
	}
}

func TestInputBuiltinReturnsUserInput(t *testing.T) {
	source := `
first = input()
second = input("second: ")

print(first)
print(second)
`

	var l lexer.Lexer
	l.Set(source)
	l.Tokenize()

	var p parser.Parser
	p.Set(l.Get())
	p.Parse()

	runtime := New()
	runtime.SetInput(strings.NewReader("hello\nworld\n"))
	result, err := runtime.Execute(ir.Lower(p.Get()))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	want := []string{"hello", "world"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("output = %#v, want %#v", result.Output, want)
	}
}

func TestObjectIndexCanCallCommandFunction(t *testing.T) {
	source := `
command = {}
command.hello = () => {
    print("hello world")
}

cmd = input("[bx]:> ")
run = command[cmd]()
`

	var l lexer.Lexer
	l.Set(source)
	l.Tokenize()

	var p parser.Parser
	p.Set(l.Get())
	p.Parse()

	runtime := New()
	runtime.SetInput(strings.NewReader("hello\n"))
	result, err := runtime.Execute(ir.Lower(p.Get()))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	want := []string{"hello world"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("output = %#v, want %#v", result.Output, want)
	}
}

func TestStringEscapes(t *testing.T) {
	source := `
print("line one\nline two")
print("tab\tvalue")
print("quote: \"")
print("slash: \\")
`

	var l lexer.Lexer
	l.Set(source)
	l.Tokenize()

	var p parser.Parser
	p.Set(l.Get())
	p.Parse()

	runtime := New()
	result, err := runtime.Execute(ir.Lower(p.Get()))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	want := []string{"line one\nline two", "tab\tvalue", "quote: \"", "slash: \\"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("output = %#v, want %#v", result.Output, want)
	}
}

func TestNestedIfConditionDoesNotDependOnInnerTrailingComma(t *testing.T) {
	sources := []string{
		`
if(
    if(
        true,
        () => {
            return true
        },
        () => {
            return false
        }
    )

    () => {return print("im happy")},
    () => {return print("im sad")}
)
`,
		`
if(
    if(
        true,
        () => {
            return true
        },
        () => {
            return false
        },
    )

    () => {return print("im happy")},
    () => {return print("im sad")}
)
`,
	}

	for _, source := range sources {
		var l lexer.Lexer
		l.Set(source)
		l.Tokenize()

		var p parser.Parser
		p.Set(l.Get())
		p.Parse()

		runtime := New()
		result, err := runtime.Execute(ir.Lower(p.Get()))
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}

		want := []string{"im happy"}
		if !sameStrings(result.Output, want) {
			t.Fatalf("output = %#v, want %#v", result.Output, want)
		}
	}
}
