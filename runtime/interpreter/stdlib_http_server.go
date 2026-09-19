package interpreter

import (
	"blbx_lang/syntax/diagnostic"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func (r *networkResource) recordOutput(stdout, stderr string) {
	r.logMu.Lock()
	defer r.logMu.Unlock()
	for _, part := range []struct {
		target *string
		text   string
	}{{&r.output, stdout}, {&r.errorOutput, stderr}} {
		remaining := binaryLimit - len(*part.target)
		if len(part.text) > remaining {
			part.text = part.text[:remaining]
			r.truncated = true
		}
		*part.target += part.text
	}
}

func addHTTPServer(exports map[string]Value) {
	exports["server_output"] = contextNative("std.networking.server_output", diagnostic.NetworkFailure, []Kind{ResourceKind}, func(i *Interpreter, a []Value) (Value, error) {
		r, err := networkHandle(a[0], "http_server")
		if err != nil {
			return Null(), err
		}
		r.logMu.Lock()
		defer r.logMu.Unlock()
		result := Object(map[string]Value{"stdout": String(r.output), "stderr": String(r.errorOutput), "truncated": Boolean(r.truncated)})
		r.output, r.errorOutput, r.truncated = "", "", false
		return result, nil
	})
	exports["serve"] = contextNative("std.networking.serve", diagnostic.NetworkFailure, []Kind{StringKind, FunctionKind}, func(i *Interpreter, a []Value) (Value, error) {
		clone := newSnapshot()
		handler := clone.value(a[1])
		root := clone.scope(i.global)
		moduleRoot, currentFile := i.moduleRoot, i.currentFile
		args := append([]string(nil), i.Args...)
		listener, err := net.Listen("tcp", a[0].String)
		if err != nil {
			return Null(), err
		}
		resource := &networkResource{kind: "http_server", listener: listener.(*net.TCPListener), done: make(chan struct{})}
		slots := make(chan struct{}, 64)
		server := &http.Server{ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
		resource.server = server
		server.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			default:
				http.Error(w, "server busy", http.StatusServiceUnavailable)
				return
			}
			defer func() {
				if recovered := recover(); recovered != nil {
					resource.recordOutput("", fmt.Sprintf("handler failed: %v\n", recovered))
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			data, err := io.ReadAll(io.LimitReader(req.Body, stdOutputLimit+1))
			if err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			if len(data) > stdOutputLimit {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			query := map[string]Value{}
			for key, items := range req.URL.Query() {
				values := []Value{}
				for _, v := range items {
					values = append(values, String(v))
				}
				query[key] = Array(values)
			}
			request := Object(map[string]Value{"method": String(req.Method), "path": String(req.URL.Path), "url": String(req.URL.String()), "query": Object(query), "headers": headersValue(req.Header), "body": String(string(data)), "remote_address": String(req.RemoteAddr)})
			c := newSnapshot()
			fn := c.value(handler)
			worker := New()
			worker.global = c.scope(root)
			worker.moduleRoot, worker.currentFile = moduleRoot, currentFile
			worker.Args = append([]string(nil), args...)
			worker.SetInput(strings.NewReader(""))
			var stdout, stderr limitedOutput
			worker.SetOutput(&stdout)
			worker.ErrorWriter = &stderr
			result, err := worker.callValue(fn, []Value{request})
			resource.recordOutput(stdout.String(), stderr.String())
			if err != nil {
				resource.recordOutput("", diagnostic.Runtime(err).Error()+"\n")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			status, headers, body, err := serverResponse(result.value)
			if err != nil {
				resource.recordOutput("", err.Error()+"\n")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			for key, values := range headers {
				for _, v := range values {
					w.Header().Add(key, v)
				}
			}
			w.WriteHeader(status)
			w.Write(body)
		})
		go func() {
			defer close(resource.done)
			err := server.Serve(listener)
			if err != nil && err != http.ErrServerClosed {
				resource.serveErr = err
			}
		}()
		return resourceValue(resource), nil
	})
}

func serverResponse(value Value) (int, http.Header, []byte, error) {
	status := 200
	headers := http.Header{}
	bodyValue := value
	if value.Kind == ObjectKind {
		for key := range value.Object {
			if key != "status" && key != "headers" && key != "body" {
				return 0, nil, nil, fmt.Errorf("unknown response field %q", key)
			}
		}
		if code, ok := value.Object["status"]; ok {
			if code.Kind != IntegerKind || code.Integer < 200 || code.Integer > 599 {
				return 0, nil, nil, fmt.Errorf("response status must be between 200 and 599")
			}
			status = int(code.Integer)
		}
		if h, ok := value.Object["headers"]; ok {
			if err := applyHeaders(headers, h); err != nil {
				return 0, nil, nil, err
			}
		}
		bodyValue = String("")
		if body, ok := value.Object["body"]; ok {
			bodyValue = body
		}
	}
	body, err := valueBytes(bodyValue)
	return status, headers, body, err
}
