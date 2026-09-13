package interpreter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModuleImports(t *testing.T) {
	root := t.TempDir()
	stdDir := filepath.Join(root, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("mkdir std: %v", err)
	}

	writeTestFile(t, filepath.Join(stdDir, "__init__.bx"), `name = "std"`)
	writeTestFile(t, filepath.Join(stdDir, "io.bx"), `
word = "module-word"
_hidden = "secret"
echo = (value) => {
    return value
}
`)
	if _, err := os.Stat(filepath.Join(stdDir, "io.bx")); err != nil {
		t.Fatalf("missing module fixture: %v", err)
	}
	writeTestFile(t, filepath.Join(root, "main.bx"), `
import std.io as io
from std.io import word

print(io.echo("hello from import"))
print(word)
print(io._hidden)
`)

	runtime := New()
	result, err := runtime.ExecuteFile(filepath.Join(root, "main.bx"))
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	want := []string{"hello from import", "module-word", "null"}
	if !sameStrings(result.Output, want) {
		t.Fatalf("output = %#v, want %#v", result.Output, want)
	}
}

func writeTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
