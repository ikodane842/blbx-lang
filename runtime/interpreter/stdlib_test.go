package interpreter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStandardLibraryImports(t *testing.T) {
	result, err := runClassSource(t, `
import std.strings as text
from std.math import sqrt
import std.collections as collections
import std.serialization as json
import std.time as time
import std.processes as processes
print(text.join(text.split("a,b", ","), "-"))
print(sqrt(81))
print(collections.range(4).join(","))
print(collections.keys({"z": 1, "a": 2}).join(","))
print(json.json_encode(json.json_decode("{\"a\":[1,true,null]}")))
print(time.format(time.parse("2026-01-01T00:00:00Z")))
print(typeof(processes.cwd()))
`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a-b", "9", "0,1,2,3", "a,z", `{"a":[1,true,null]}`, "2026-01-01T00:00:00Z", "string"}
	if !sameStrings(result.Output, want) {
		t.Fatal(result.Output)
	}
}

func invokeStd(t *testing.T, module, method string, args ...Value) (Value, error) {
	t.Helper()
	value, ok := standardModule("std." + module)
	if !ok {
		t.Fatal(module)
	}
	function, ok := value.Object[method]
	if !ok {
		t.Fatal(method)
	}
	result, err := New().callValue(function, args)
	return result.value, err
}

func TestStandardFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if _, err := invokeStd(t, "files", "write", String(path), String("hello")); err != nil {
		t.Fatal(err)
	}
	if _, err := invokeStd(t, "files", "append", String(path), String(" world")); err != nil {
		t.Fatal(err)
	}
	value, err := invokeStd(t, "files", "read", String(path))
	if err != nil || value.String != "hello world" {
		t.Fatal(value, err)
	}
	value, err = invokeStd(t, "files", "list", String(root))
	if err != nil || len(value.Array) != 1 {
		t.Fatal(value, err)
	}
	value, err = invokeStd(t, "files", "exists", String(path+".missing"))
	if err != nil || value.Boolean {
		t.Fatal(value, err)
	}
	if _, err = invokeStd(t, "files", "read", String(path+".missing")); err == nil {
		t.Fatal("expected read error")
	}
}

func TestStandardNetworking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "ok")
		if r.Method == "POST" {
			b, _ := io.ReadAll(r.Body)
			w.Write(b)
			return
		}
		w.WriteHeader(404)
		fmt.Fprint(w, "missing")
	}))
	defer server.Close()
	value, err := invokeStd(t, "networking", "get", String(server.URL))
	if err != nil || value.Object["status"].Integer != 404 || value.Object["body"].String != "missing" {
		t.Fatal(value, err)
	}
	value, err = invokeStd(t, "networking", "post", String(server.URL), String("hello"), String("text/plain"))
	if err != nil || value.Object["body"].String != "hello" {
		t.Fatal(value, err)
	}
	if _, err := invokeStd(t, "networking", "get", String("file:///bad")); err == nil {
		t.Fatal("expected scheme error")
	}
}

func TestStdProcessHelper(t *testing.T) {
	if os.Getenv("BLBX_STDLIB_HELPER") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, "hello")
	fmt.Fprint(os.Stderr, "diagnostic")
	os.Exit(7)
}

func TestStandardProcesses(t *testing.T) {
	t.Setenv("BLBX_STDLIB_HELPER", "1")
	value, err := invokeStd(t, "processes", "run", String(os.Args[0]), Array([]Value{String("-test.run=^TestStdProcessHelper$")}))
	if err != nil || value.Object["exit_code"].Integer != 7 || value.Object["stdout"].String != "hello" || value.Object["stderr"].String != "diagnostic" {
		t.Fatal(value, err)
	}
}

func TestStandardLibraryErrors(t *testing.T) {
	for _, tc := range []struct {
		module, method string
		args           []Value
	}{
		{"math", "sqrt", nil}, {"math", "sqrt", []Value{String("9")}}, {"math", "sqrt", []Value{Integer(-1)}},
		{"serialization", "json_decode", []Value{String("1 2")}}, {"serialization", "json_decode", []Value{String("9223372036854775808")}},
		{"serialization", "json_encode", []Value{FunctionValue(&Function{})}},
		{"collections", "range", []Value{Integer(-1)}}, {"time", "sleep", []Value{Integer(-1)}},
		{"processes", "run", []Value{String("unused"), Array([]Value{Integer(1)})}},
	} {
		if _, err := invokeStd(t, tc.module, tc.method, tc.args...); err == nil {
			t.Errorf("%s.%s should fail", tc.module, tc.method)
		}
	}
	cycle := Object(map[string]Value{})
	cycle.Object["self"] = cycle
	if _, err := invokeStd(t, "serialization", "json_encode", cycle); err == nil {
		t.Fatal("cycle accepted")
	}
	value, err := invokeStd(t, "serialization", "json_decode", String("9223372036854775807"))
	if err != nil || value.Integer != 9223372036854775807 {
		t.Fatal(value, err)
	}
}

func TestStandardFileImportIntegration(t *testing.T) {
	// Import integration exercises an ordinary BLBX module call with a path
	// containing OS-specific separators and quotes, without shell escaping.
	path := filepath.Join(t.TempDir(), "roundtrip.txt")
	quoted, _ := json.Marshal(path)
	result, err := runClassSource(t, "import std.files as files\nfiles.write("+string(quoted)+", \"ok\")\nprint(files.read("+string(quoted)+"))")
	if err != nil || strings.Join(result.Output, "") != "ok" {
		t.Fatal(result, err)
	}
}
