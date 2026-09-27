// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"blbx_lang/syntax/diagnostic"
	"fmt"
	"io"
	"os"
	goruntime "runtime"
	"strings"
)

func addOS(add nativeAdder, exports map[string]Value) {
	exports["platform"], exports["arch"] = String(goruntime.GOOS), String(goruntime.GOARCH)
	add("cwd", nil, func([]Value) (Value, error) { v, e := os.Getwd(); return String(v), e })
	add("home", nil, func([]Value) (Value, error) { v, e := os.UserHomeDir(); return String(v), e })
	add("temp_dir", nil, func([]Value) (Value, error) { return String(os.TempDir()), nil })
	add("env", []Kind{StringKind}, func(a []Value) (Value, error) {
		v, ok := os.LookupEnv(a[0].String)
		if !ok {
			return Null(), nil
		}
		return String(v), nil
	})
	add("set_env", []Kind{StringKind, StringKind}, func(a []Value) (Value, error) { return Null(), os.Setenv(a[0].String, a[1].String) })
	add("unset_env", []Kind{StringKind}, func(a []Value) (Value, error) { return Null(), os.Unsetenv(a[0].String) })
	add("environment", nil, func([]Value) (Value, error) {
		values := map[string]Value{}
		for _, entry := range os.Environ() {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) == 2 {
				values[parts[0]] = String(parts[1])
			}
		}
		return Object(values), nil
	})
	exports["args"] = contextNative("std.os.args", diagnostic.OSFailure, nil, func(i *VM, a []Value) (Value, error) {
		values := []Value{}
		for _, v := range i.Args {
			values = append(values, String(v))
		}
		return Array(values), nil
	})
	exports["read_line"] = contextNative("std.os.read_line", diagnostic.OSFailure, nil, func(i *VM, a []Value) (Value, error) {
		var line []byte
		for {
			part, prefix, err := i.Input.ReadLine()
			if err != nil && err != io.EOF {
				return Null(), err
			}
			if len(line)+len(part) > stdOutputLimit {
				return Null(), fmt.Errorf("input line exceeds 8 MiB")
			}
			line = append(line, part...)
			if err == io.EOF && len(line) == 0 {
				return Null(), nil
			}
			if !prefix {
				return String(string(line)), nil
			}
		}
	})
	for _, name := range []string{"write_stdout", "write_stderr"} {
		n := name
		exports[n] = contextNative("std.os."+n, diagnostic.OSFailure, []Kind{StringKind}, func(i *VM, a []Value) (Value, error) {
			writer := i.Writer
			if n == "write_stderr" {
				writer = i.ErrorWriter
			}
			_, err := io.WriteString(writer, a[0].String)
			return Null(), err
		})
	}
	add("run", []Kind{StringKind, ArrayKind}, runProcess)
}
