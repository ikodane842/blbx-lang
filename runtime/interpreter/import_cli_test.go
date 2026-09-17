package interpreter

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportCLIDemo(t *testing.T) {
	runtime := New()
	runtime.SetOutput(io.Discard)
	result, err := runtime.ExecuteFile(filepath.Join("..", "..", "import_cli", "demo.bx"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Output) != 5 || result.Output[1] != "hello from BLBX imports" || result.Output[2] != "42" {
		t.Fatal(result.Output)
	}
	if !strings.Contains(result.Output[3], `"text":"two words"`) || !strings.Contains(result.Output[4], `"name":"echo"`) {
		t.Fatal(result.Output)
	}
}

func TestImportCLIContinuesAfterBadCommands(t *testing.T) {
	runtime := New()
	runtime.SetOutput(io.Discard)
	runtime.SetInput(strings.NewReader("unknown\nadd x 2\necho \"unterminated\necho \"still running\"\nexit\n"))
	result, err := runtime.ExecuteFile(filepath.Join("..", "..", "import_cli", "main.bx"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"BLBX import demo. Type help; exit or an empty line closes the CLI.", "Unknown command: unknown", "add expects two valid numbers.", "Unterminated quoted string.", "still running"}
	if !sameStrings(result.Output, want) {
		t.Fatal(result.Output)
	}
}

func TestImportDirectoryRequiresInit(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	if err := os.Mkdir(pkg, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(pkg, "child.bx"), `_name = "visible"`)
	main := filepath.Join(root, "main.bx")
	writeTestFile(t, main, `from pkg.child import _name print(_name)`)
	if _, err := New().ExecuteFile(main); err == nil || !strings.Contains(err.Error(), "__init__.bx") {
		t.Fatalf("expected missing marker error, got %v", err)
	}
	writeTestFile(t, filepath.Join(pkg, "__init__.bx"), "")
	runtime := New()
	runtime.SetOutput(io.Discard)
	result, err := runtime.ExecuteFile(main)
	if err != nil || !sameStrings(result.Output, []string{"visible"}) {
		t.Fatal(result.Output, err)
	}
}
