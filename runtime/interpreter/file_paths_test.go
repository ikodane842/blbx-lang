package interpreter

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFilesResolveBesideImportingScript(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "hello.txt"), "entry")
	writeTestFile(t, filepath.Join(sub, "__init__.bx"), "")
	writeTestFile(t, filepath.Join(sub, "hello.txt"), "module")
	writeTestFile(t, filepath.Join(sub, "reader.bx"), `
from std.files import read
get = () => { return read("./hello.txt") }
`)
	main := filepath.Join(root, "main.bx")
	writeTestFile(t, main, `
import std.files as files
import sub.reader as reader
print(files.read("hello.txt"))
print(files.read("./hello.txt"))
print(reader.get())
files.mkdir("output")
files.write("output/result.txt", "one")
files.append("output/result.txt", "two")
print(files.exists("output/result.txt"))
print(files.list("output").join())
`)
	runtime := New()
	runtime.SetOutput(io.Discard)
	result, err := runtime.ExecuteFile(main)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(result.Output, []string{"entry", "entry", "module", "true", "result.txt"}) {
		t.Fatal(result.Output)
	}
	data, err := os.ReadFile(filepath.Join(root, "output", "result.txt"))
	if err != nil || string(data) != "onetwo" {
		t.Fatal(string(data), err)
	}
}
