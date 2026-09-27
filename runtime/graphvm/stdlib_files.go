// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func readBounded(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, stdOutputLimit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > stdOutputLimit {
		return nil, fmt.Errorf("file exceeds 8 MiB")
	}
	return data, nil
}

func addFileUtilities(add nativeAdder, resolve func(string) string) {
	add("read_bytes", []Kind{StringKind}, func(a []Value) (Value, error) {
		b, e := readBounded(resolve(a[0].String))
		if e != nil {
			return Null(), e
		}
		if len(b) > binaryLimit {
			return Null(), fmt.Errorf("binary file exceeds 1 MiB")
		}
		return byteArray(b), nil
	})
	add("write_bytes", []Kind{StringKind, ArrayKind}, func(a []Value) (Value, error) {
		b, e := valueBytes(a[1])
		if e != nil {
			return Null(), e
		}
		return Null(), os.WriteFile(resolve(a[0].String), b, 0644)
	})
	add("remove", []Kind{StringKind}, func(a []Value) (Value, error) { return Null(), os.Remove(resolve(a[0].String)) })
	add("rename", []Kind{StringKind, StringKind}, func(a []Value) (Value, error) { return Null(), os.Rename(resolve(a[0].String), resolve(a[1].String)) })
	add("copy", []Kind{StringKind, StringKind}, func(a []Value) (Value, error) {
		source, err := os.Open(resolve(a[0].String))
		if err != nil {
			return Null(), err
		}
		defer source.Close()
		info, err := source.Stat()
		if err != nil {
			return Null(), err
		}
		if !info.Mode().IsRegular() {
			return Null(), fmt.Errorf("copy source must be a regular file")
		}
		path := resolve(a[1].String)
		dest, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if err != nil {
			return Null(), err
		}
		count, err := io.Copy(dest, source)
		closeErr := dest.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			os.Remove(path)
			return Null(), err
		}
		return Integer(count), nil
	})
	add("stat", []Kind{StringKind}, func(a []Value) (Value, error) {
		info, err := os.Stat(resolve(a[0].String))
		if err != nil {
			return Null(), err
		}
		return Object(map[string]Value{"name": String(info.Name()), "size": Integer(info.Size()), "is_dir": Boolean(info.IsDir()), "is_file": Boolean(info.Mode().IsRegular()), "modified_ms": Integer(info.ModTime().UnixMilli()), "mode": Integer(int64(info.Mode().Perm()))}), nil
	})
	for _, name := range []string{"is_file", "is_dir"} {
		n := name
		add(n, []Kind{StringKind}, func(a []Value) (Value, error) {
			info, err := os.Stat(resolve(a[0].String))
			if os.IsNotExist(err) {
				return Boolean(false), nil
			}
			if err != nil {
				return Null(), err
			}
			if n == "is_dir" {
				return Boolean(info.IsDir()), nil
			}
			return Boolean(info.Mode().IsRegular()), nil
		})
	}
	add("absolute", []Kind{StringKind}, func(a []Value) (Value, error) { p, e := filepath.Abs(resolve(a[0].String)); return String(p), e })
	add("relative", []Kind{StringKind, StringKind}, func(a []Value) (Value, error) {
		base, e := filepath.Abs(resolve(a[0].String))
		if e != nil {
			return Null(), e
		}
		target, e := filepath.Abs(resolve(a[1].String))
		if e != nil {
			return Null(), e
		}
		p, e := filepath.Rel(base, target)
		return String(p), e
	})
	for name, operation := range map[string]func(string) string{"basename": filepath.Base, "dirname": filepath.Dir, "extension": filepath.Ext, "clean": filepath.Clean} {
		fn := operation
		add(name, []Kind{StringKind}, func(a []Value) (Value, error) { return String(fn(a[0].String)), nil })
	}
	add("is_absolute", []Kind{StringKind}, func(a []Value) (Value, error) { return Boolean(filepath.IsAbs(a[0].String)), nil })
	add("walk", []Kind{StringKind}, func(a []Value) (Value, error) {
		root, e := filepath.Abs(resolve(a[0].String))
		if e != nil {
			return Null(), e
		}
		paths := []Value{}
		e = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if len(paths) >= 100000 {
				return fmt.Errorf("walk exceeds 100000 entries")
			}
			paths = append(paths, String(path))
			return nil
		})
		return Array(paths), e
	})
}
