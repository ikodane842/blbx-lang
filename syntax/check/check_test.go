package check

import (
	"blbx_lang/syntax/lexer"
	"blbx_lang/syntax/parser"
	"strings"
	"testing"
)

func TestValidSource(t *testing.T) {
	for _, source := range []string{
		"", "// comment", "\uFEFFname = 1\r\n",
		`f = (a, b) => { return (a).add(b) } print(f(2, 5))`,
		`x = [1, 2,] obj = {} obj.name = "x" obj["name"] = "y"`,
		`import std.io as io from std.io import read as r, write`,
		`if(true () => {return true} () => {return false})`,
		`x = /* comment */ 1 print(/* comment */ x)`,
		"x = \"a\nb\"\nprint(x)",
		`a = b = 3 print(2 * 3)`,
	} {
		_, found := Source("test.bx", source)
		if len(found) != 0 {
			t.Errorf("%q: %v", source, found)
		}
	}
}

func TestInvalidSource(t *testing.T) {
	for _, tc := range []struct{ source, code string }{
		{"@", "BX1001"}, {`"open`, "BX1002"}, {"/* open", "BX1003"},
		{"\xff", "BX1004"}, {"x =", "BX2002"}, {"return", "BX2002"},
		{"print(", "BX2001"}, {"[1 2]", "BX2001"}, {"}", "BX2002"},
		{"1 = 2", "BX2003"}, {"x = 2 *", "BX2002"},
		{"import", "BX2001"}, {"import a.", "BX2001"}, {"import a as", "BX2001"},
		{"from a", "BX2001"}, {"from a import", "BX2001"}, {"from a import b,", "BX2001"},
		{strings.Repeat("(", 300), "BX2004"},
		{strings.Repeat("a = ", 300) + "1", "BX2004"},
	} {
		_, found := Source("test.bx", tc.source)
		if len(found) == 0 || found[0].Code != tc.code {
			t.Errorf("%q: got %v, want %s", tc.source, found, tc.code)
		}
	}
}

func TestPositionsAndRecovery(t *testing.T) {
	_, found := Source("test.bx", "x = \"a\nb\"\r\n😀\nprint(\n")
	if len(found) != 2 {
		t.Fatalf("got %v", found)
	}
	if found[0].Line != 3 || found[0].Column != 1 || found[0].EndColumn != 2 {
		t.Errorf("Unicode location: %+v", found[0])
	}
	if found[1].Line != 5 || found[1].Column != 1 {
		t.Errorf("EOF location: %+v", found[1])
	}
	_, found = Source("test.bx", "a = )\nb = ]\nc = 1")
	if len(found) != 2 {
		t.Fatalf("expected two independent syntax errors: %v", found)
	}
}

func TestLexerParserReuse(t *testing.T) {
	var l lexer.Lexer
	var p parser.Parser
	for _, source := range []string{"@", "x = 1", ""} {
		l.Set(source)
		l.Tokenize()
		p.Set(l.Get())
		p.Parse()
		if source != "@" && (len(l.Diagnostics) != 0 || len(p.Diagnostics) != 0) {
			t.Fatal("stale diagnostics")
		}
	}
}

func FuzzSource(f *testing.F) {
	for _, seed := range []string{"", "a = [1, 2]", "print(\"hello\")", "/*", "x =", "from x import y", "f = () => {}", "([)]"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 16384 {
			t.Skip()
		}
		_, found := Source("fuzz.bx", source)
		for _, d := range found {
			if d.Line < 1 || d.Column < 1 || d.EndLine < d.Line || (d.EndLine == d.Line && d.EndColumn < d.Column) {
				t.Fatalf("invalid range: %+v", d)
			}
		}
	})
}
