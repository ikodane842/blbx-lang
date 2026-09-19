package interpreter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func addJSONUtilities(add nativeAdder, resolve func(string) string) {
	add("json_valid", []Kind{StringKind}, func(a []Value) (Value, error) { _, err := decodeJSON(a); return Boolean(err == nil), nil })
	add("json_pretty", []Kind{"", IntegerKind}, func(a []Value) (Value, error) {
		if a[1].Integer < 0 || a[1].Integer > 8 {
			return Null(), fmt.Errorf("indent must be between 0 and 8 spaces")
		}
		value, err := encodeJSON(a[:1])
		if err != nil {
			return Null(), err
		}
		var out bytes.Buffer
		err = json.Indent(&out, []byte(value.String), "", strings.Repeat(" ", int(a[1].Integer)))
		if out.Len() > stdOutputLimit {
			return Null(), fmt.Errorf("JSON output exceeds 8 MiB")
		}
		return String(out.String()), err
	})
	add("json_read", []Kind{StringKind}, func(a []Value) (Value, error) {
		data, err := readBounded(resolve(a[0].String))
		if err != nil {
			return Null(), err
		}
		return decodeJSON([]Value{String(string(data))})
	})
	add("json_write", []Kind{StringKind, ""}, func(a []Value) (Value, error) {
		data, err := encodeJSON(a[1:])
		if err != nil {
			return Null(), err
		}
		return Null(), os.WriteFile(resolve(a[0].String), []byte(data.String), 0644)
	})
}
