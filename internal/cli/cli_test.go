package cli

import (
	"blbx_lang/syntax/diagnostic"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func invoke(args []string, input string) (int, string, string) {
	var out, err bytes.Buffer
	code := Run(args, strings.NewReader(input), &out, &err)
	return code, out.String(), err.String()
}

func TestCheckJSON(t *testing.T) {
	code, output, stderr := invoke([]string{"check", "--format", "json", "--stdin-filename", "draft.bx", "-"}, "x = )\ny = ]")
	var found []diagnostic.Diagnostic
	if err := json.Unmarshal([]byte(output), &found); err != nil {
		t.Fatal(err)
	}
	if code != 1 || stderr != "" || len(found) != 2 || found[0].File != "draft.bx" {
		t.Fatalf("%d %s %s", code, output, stderr)
	}
	code, output, _ = invoke([]string{"check", "--format", "json", "-"}, `print(input()) import missing.module`)
	if code != 0 || strings.TrimSpace(output) != "[]" {
		t.Fatalf("check should not execute or resolve imports: %d %s", code, output)
	}
}

func TestUsageAndIO(t *testing.T) {
	for _, args := range [][]string{{"wat"}, {"check"}, {"check", "--format", "bad", "-"}, {"check", "-", "-"}, {"run"}} {
		code, _, stderr := invoke(args, "")
		if code != 2 || stderr == "" {
			t.Errorf("%v: %d %s", args, code, stderr)
		}
	}
	for _, args := range [][]string{nil, {"help"}, {"version"}} {
		code, output, _ := invoke(args, "")
		if code != 0 || output == "" {
			t.Errorf("%v: %d %s", args, code, output)
		}
	}
	missing := filepath.Join(t.TempDir(), "missing.bx")
	code, output, _ := invoke([]string{"check", "--format", "json", missing, "-"}, "x = )")
	var found []diagnostic.Diagnostic
	if err := json.Unmarshal([]byte(output), &found); err != nil {
		t.Fatal(err)
	}
	if code != 2 || len(found) != 2 || found[0].Code != "BX0001" {
		t.Fatalf("%d %s", code, output)
	}
}

func TestRunAndImports(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.bx")
	module := filepath.Join(root, "mod.bx")
	write := func(path, source string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(module, `word = "hello"`)
	write(path, `import mod print(mod.word) print(input("name: "))`)
	code, output, stderr := invoke([]string{"run", path}, "Ada\n")
	if code != 0 || output != "hello\nname: Ada\n" || stderr != "" {
		t.Fatalf("%d %q %s", code, output, stderr)
	}
	write(path, `print("must not execute") x = )`)
	code, output, stderr = invoke([]string{"run", path}, "")
	if code != 1 || output != "" || !strings.Contains(stderr, "BX2002") {
		t.Fatalf("%d %q %s", code, output, stderr)
	}
	write(path, `import mod`)
	write(module, `x = )`)
	code, _, stderr = invoke([]string{"run", path}, "")
	if code != 1 || !strings.Contains(stderr, "mod.bx") || !strings.Contains(stderr, "BX2002") {
		t.Fatalf("%d %s", code, stderr)
	}
}

// func is an ordinary BLBX identifier, including when it names a function.
func TestFuncIsNotReserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "func.bx")
	for _, source := range []string{
		`func = () => {}`,
		`func = () => {} func()`,
		`func = () => { return 42 } print(func())`,
	} {
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		code, output, stderr := invoke([]string{"check", "--format", "json", path}, "")
		if code != 0 || strings.TrimSpace(output) != "[]" || stderr != "" {
			t.Fatalf("check %q: %d %q %s", source, code, output, stderr)
		}
		code, output, stderr = invoke([]string{"run", path}, "")
		if code != 0 || stderr != "" {
			t.Fatalf("run %q: %d %q %s", source, code, output, stderr)
		}
		if strings.Contains(source, "42") && output != "42\n" {
			t.Fatalf("unexpected output: %q", output)
		}
	}
}
