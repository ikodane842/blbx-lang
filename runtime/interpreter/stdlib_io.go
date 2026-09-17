package interpreter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const stdOutputLimit = 8 << 20

func encodeJSON(args []Value) (Value, error) {
	v, err := plainJSON(args[0], 0)
	if err != nil {
		return Null(), err
	}
	b, err := json.Marshal(v)
	return String(string(b)), err
}

func plainJSON(v Value, depth int) (interface{}, error) {
	if depth > 128 {
		return nil, fmt.Errorf("cyclic value or nesting exceeds 128")
	}
	switch v.Kind {
	case NullKind:
		return nil, nil
	case StringKind:
		return v.String, nil
	case BooleanKind:
		return v.Boolean, nil
	case IntegerKind:
		return v.Integer, nil
	case FloatKind:
		return v.Float, nil
	case ArrayKind:
		out := make([]interface{}, len(v.Array))
		for j, item := range v.Array {
			value, err := plainJSON(item, depth+1)
			if err != nil {
				return nil, err
			}
			out[j] = value
		}
		return out, nil
	case ObjectKind:
		out := map[string]interface{}{}
		for key, item := range v.Object {
			value, err := plainJSON(item, depth+1)
			if err != nil {
				return nil, err
			}
			out[key] = value
		}
		return out, nil
	default:
		return nil, fmt.Errorf("cannot serialize %s", v.Kind)
	}
}

func decodeJSON(args []Value) (Value, error) {
	decoder := json.NewDecoder(strings.NewReader(args[0].String))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return Null(), err
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return Null(), fmt.Errorf("expected exactly one JSON value")
	}
	return fromJSON(value, 0)
}

func fromJSON(v interface{}, depth int) (Value, error) {
	if depth > 128 {
		return Null(), fmt.Errorf("JSON nesting exceeds 128")
	}
	switch x := v.(type) {
	case nil:
		return Null(), nil
	case bool:
		return Boolean(x), nil
	case string:
		return String(x), nil
	case json.Number:
		if !strings.ContainsAny(string(x), ".eE") {
			n, err := strconv.ParseInt(string(x), 10, 64)
			return Integer(n), err
		}
		n, err := x.Float64()
		if err != nil {
			return Null(), err
		}
		return finiteNumber(n)
	case []interface{}:
		out := make([]Value, len(x))
		for j, item := range x {
			value, err := fromJSON(item, depth+1)
			if err != nil {
				return Null(), err
			}
			out[j] = value
		}
		return Array(out), nil
	case map[string]interface{}:
		out := map[string]Value{}
		for key, item := range x {
			value, err := fromJSON(item, depth+1)
			if err != nil {
				return Null(), err
			}
			out[key] = value
		}
		return Object(out), nil
	}
	return Null(), fmt.Errorf("unsupported JSON value")
}

func httpGet(a []Value) (Value, error) { return httpRequest("GET", a[0].String, "", "") }
func httpPost(a []Value) (Value, error) {
	return httpRequest("POST", a[0].String, a[1].String, a[2].String)
}

func httpRequest(method, url, body, contentType string) (Value, error) {
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return Null(), err
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" {
		return Null(), fmt.Errorf("URL must use http or https")
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return Null(), err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, stdOutputLimit+1))
	if err != nil {
		return Null(), err
	}
	if len(data) > stdOutputLimit {
		return Null(), fmt.Errorf("response exceeds 8 MiB")
	}
	headers := map[string]Value{}
	for key, values := range response.Header {
		items := []Value{}
		for _, value := range values {
			items = append(items, String(value))
		}
		headers[strings.ToLower(key)] = Array(items)
	}
	return Object(map[string]Value{"status": Integer(int64(response.StatusCode)), "body": String(string(data)), "headers": Object(headers)}), nil
}

type limitedOutput struct {
	bytes.Buffer
	err error
}

func (b *limitedOutput) Write(p []byte) (int, error) {
	if len(p) > stdOutputLimit-b.Len() {
		b.err = fmt.Errorf("process output exceeds 8 MiB")
		return 0, b.err
	}
	return b.Buffer.Write(p)
}

func runProcess(a []Value) (Value, error) {
	args := []string{}
	for _, value := range a[1].Array {
		if value.Kind != StringKind {
			return Null(), fmt.Errorf("process arguments must be strings")
		}
		args = append(args, value.String)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, a[0].String, args...)
	command.WaitDelay = time.Second
	var stdout, stderr limitedOutput
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if stdout.err != nil {
		return Null(), stdout.err
	}
	if stderr.err != nil {
		return Null(), stderr.err
	}
	if ctx.Err() != nil {
		return Null(), fmt.Errorf("process timed out after 15 seconds")
	}
	code := int64(0)
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = int64(exit.ExitCode())
		} else {
			return Null(), err
		}
	}
	return Object(map[string]Value{"exit_code": Integer(code), "stdout": String(stdout.String()), "stderr": String(stderr.String())}), nil
}
