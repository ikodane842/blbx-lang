// Adapted from the BLBX interpreter to operate on compiled graph components.
package graphvm

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Resource handles represent shared external I/O, not copied BLBX state.
// Read/write locks serialize per-direction deadlines; Close interrupts I/O.
type networkResource struct {
	kind                string
	conn                net.Conn
	listener            *net.TCPListener
	packet              net.PacketConn
	server              *http.Server
	done                chan struct{}
	serveErr            error
	once                sync.Once
	closeErr            error
	readMu, writeMu     sync.Mutex
	logMu               sync.Mutex
	output, errorOutput string
	truncated           bool
}

func resourceValue(r *networkResource) Value { return Value{Kind: ResourceKind, Resource: r} }
func networkHandle(v Value, kind string) (*networkResource, error) {
	if v.Resource == nil || (kind != "" && v.Resource.kind != kind) {
		return nil, fmt.Errorf("expected %s network resource", kind)
	}
	return v.Resource, nil
}
func timeoutDuration(v Value) (time.Duration, error) {
	if v.Kind != IntegerKind || v.Integer < 1 || v.Integer > 300000 {
		return 0, fmt.Errorf("timeout_ms must be an integer from 1 to 300000")
	}
	return time.Duration(v.Integer) * time.Millisecond, nil
}
func receiveLimit(v Value) (int, error) {
	if v.Integer < 1 || v.Integer > binaryLimit {
		return 0, fmt.Errorf("max_bytes must be between 1 and 1048576")
	}
	return int(v.Integer), nil
}

func addNetworking(add nativeAdder, exports map[string]Value) {
	add("request", []Kind{StringKind, StringKind, ""}, requestHTTP)
	add("tcp_connect", []Kind{StringKind, IntegerKind}, func(a []Value) (Value, error) {
		timeout, e := timeoutDuration(a[1])
		if e != nil {
			return Null(), e
		}
		conn, e := net.DialTimeout("tcp", a[0].String, timeout)
		if e != nil {
			return Null(), e
		}
		return resourceValue(&networkResource{kind: "tcp", conn: conn}), nil
	})
	add("tcp_listen", []Kind{StringKind}, func(a []Value) (Value, error) {
		listener, e := net.Listen("tcp", a[0].String)
		if e != nil {
			return Null(), e
		}
		return resourceValue(&networkResource{kind: "tcp_listener", listener: listener.(*net.TCPListener)}), nil
	})
	add("tcp_accept", []Kind{ResourceKind, IntegerKind}, func(a []Value) (Value, error) {
		r, e := networkHandle(a[0], "tcp_listener")
		if e != nil {
			return Null(), e
		}
		timeout, e := timeoutDuration(a[1])
		if e != nil {
			return Null(), e
		}
		r.readMu.Lock()
		defer r.readMu.Unlock()
		if e = r.listener.SetDeadline(time.Now().Add(timeout)); e != nil {
			return Null(), e
		}
		conn, e := r.listener.Accept()
		if e != nil {
			return Null(), e
		}
		return resourceValue(&networkResource{kind: "tcp", conn: conn}), nil
	})
	add("tcp_send", []Kind{ResourceKind, "", IntegerKind}, func(a []Value) (Value, error) {
		r, e := networkHandle(a[0], "tcp")
		if e != nil {
			return Null(), e
		}
		data, e := valueBytes(a[1])
		if e != nil {
			return Null(), e
		}
		timeout, e := timeoutDuration(a[2])
		if e != nil {
			return Null(), e
		}
		r.writeMu.Lock()
		defer r.writeMu.Unlock()
		if e = r.conn.SetWriteDeadline(time.Now().Add(timeout)); e != nil {
			return Null(), e
		}
		n, e := r.conn.Write(data)
		if e == nil && n != len(data) {
			e = io.ErrShortWrite
		}
		return Integer(int64(n)), e
	})
	add("tcp_receive", []Kind{ResourceKind, IntegerKind, IntegerKind}, func(a []Value) (Value, error) {
		r, e := networkHandle(a[0], "tcp")
		if e != nil {
			return Null(), e
		}
		size, e := receiveLimit(a[1])
		if e != nil {
			return Null(), e
		}
		timeout, e := timeoutDuration(a[2])
		if e != nil {
			return Null(), e
		}
		r.readMu.Lock()
		defer r.readMu.Unlock()
		if e = r.conn.SetReadDeadline(time.Now().Add(timeout)); e != nil {
			return Null(), e
		}
		data := make([]byte, size)
		n, e := r.conn.Read(data)
		eof := e == io.EOF
		if e != nil && !eof {
			return Null(), e
		}
		return Object(map[string]Value{"data": byteArray(data[:n]), "eof": Boolean(eof)}), nil
	})
	add("udp_bind", []Kind{StringKind}, func(a []Value) (Value, error) {
		conn, e := net.ListenPacket("udp", a[0].String)
		if e != nil {
			return Null(), e
		}
		return resourceValue(&networkResource{kind: "udp", packet: conn}), nil
	})
	add("udp_send", []Kind{ResourceKind, StringKind, "", IntegerKind}, func(a []Value) (Value, error) {
		r, e := networkHandle(a[0], "udp")
		if e != nil {
			return Null(), e
		}
		data, e := valueBytes(a[2])
		if e != nil {
			return Null(), e
		}
		if len(data) > 65507 {
			return Null(), fmt.Errorf("UDP payload exceeds 65507 bytes")
		}
		timeout, e := timeoutDuration(a[3])
		if e != nil {
			return Null(), e
		}
		r.writeMu.Lock()
		defer r.writeMu.Unlock()
		deadline := time.Now().Add(timeout)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()
		host, port, e := net.SplitHostPort(a[1].String)
		if e != nil {
			return Null(), e
		}
		number, e := strconv.Atoi(port)
		if e != nil || number < 1 || number > 65535 {
			return Null(), fmt.Errorf("invalid UDP port")
		}
		ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
		if e != nil {
			return Null(), e
		}
		if len(ips) == 0 {
			return Null(), fmt.Errorf("host has no addresses")
		}
		address := &net.UDPAddr{IP: ips[0].IP, Zone: ips[0].Zone, Port: number}
		if e = r.packet.SetWriteDeadline(deadline); e != nil {
			return Null(), e
		}
		n, e := r.packet.WriteTo(data, address)
		return Integer(int64(n)), e
	})
	add("udp_receive", []Kind{ResourceKind, IntegerKind, IntegerKind}, func(a []Value) (Value, error) {
		r, e := networkHandle(a[0], "udp")
		if e != nil {
			return Null(), e
		}
		size, e := receiveLimit(a[1])
		if e != nil {
			return Null(), e
		}
		timeout, e := timeoutDuration(a[2])
		if e != nil {
			return Null(), e
		}
		r.readMu.Lock()
		defer r.readMu.Unlock()
		if e = r.packet.SetReadDeadline(time.Now().Add(timeout)); e != nil {
			return Null(), e
		}
		data := make([]byte, 65536)
		n, address, e := r.packet.ReadFrom(data)
		if e != nil {
			return Null(), e
		}
		if n > size {
			return Null(), fmt.Errorf("UDP datagram exceeds max_bytes")
		}
		return Object(map[string]Value{"data": byteArray(data[:n]), "address": String(address.String())}), nil
	})
	add("address", []Kind{ResourceKind}, func(a []Value) (Value, error) {
		r, e := networkHandle(a[0], "")
		if e != nil {
			return Null(), e
		}
		if r.listener != nil {
			return String(r.listener.Addr().String()), nil
		}
		if r.conn != nil {
			return String(r.conn.LocalAddr().String()), nil
		}
		return String(r.packet.LocalAddr().String()), nil
	})
	add("close", []Kind{ResourceKind}, func(a []Value) (Value, error) {
		r, e := networkHandle(a[0], "")
		if e != nil {
			return Null(), e
		}
		r.once.Do(func() {
			switch {
			case r.server != nil:
				r.closeErr = r.server.Close()
				r.listener.Close()
			case r.listener != nil:
				r.closeErr = r.listener.Close()
			case r.conn != nil:
				r.closeErr = r.conn.Close()
			case r.packet != nil:
				r.closeErr = r.packet.Close()
			}
		})
		return Null(), r.closeErr
	})
	add("wait", []Kind{ResourceKind}, func(a []Value) (Value, error) {
		r, e := networkHandle(a[0], "http_server")
		if e != nil {
			return Null(), e
		}
		<-r.done
		return Null(), r.serveErr
	})
	add("resolve_host", []Kind{StringKind, IntegerKind}, func(a []Value) (Value, error) {
		timeout, e := timeoutDuration(a[1])
		if e != nil {
			return Null(), e
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		ips, e := net.DefaultResolver.LookupHost(ctx, a[0].String)
		if e != nil {
			return Null(), e
		}
		values := []Value{}
		for _, ip := range ips {
			values = append(values, String(ip))
		}
		return Array(values), nil
	})
	add("url_encode", []Kind{StringKind}, func(a []Value) (Value, error) { return String(url.QueryEscape(a[0].String)), nil })
	add("url_decode", []Kind{StringKind}, func(a []Value) (Value, error) { value, e := url.QueryUnescape(a[0].String); return String(value), e })
	addHTTPServer(exports)
}

func headersValue(headers map[string][]string) Value {
	values := map[string]Value{}
	for key, items := range headers {
		array := []Value{}
		for _, item := range items {
			array = append(array, String(item))
		}
		values[strings.ToLower(key)] = Array(array)
	}
	return Object(values)
}
func applyHeaders(target http.Header, value Value) error {
	if value.Kind != ObjectKind {
		return fmt.Errorf("headers must be an object")
	}
	for key, v := range value.Object {
		if key == "" {
			return fmt.Errorf("empty header name")
		}
		for _, ch := range key {
			if ch <= 32 || ch >= 127 || strings.ContainsRune("()<>@,;:\\\"/[]?={}", ch) {
				return fmt.Errorf("invalid header name %q", key)
			}
		}
		switch v.Kind {
		case StringKind:
			if strings.ContainsAny(v.String, "\r\n") {
				return fmt.Errorf("header values cannot contain line breaks")
			}
			target.Set(key, v.String)
		case ArrayKind:
			for _, item := range v.elements() {
				if item.Kind != StringKind {
					return fmt.Errorf("header values must be strings")
				}
				if strings.ContainsAny(item.String, "\r\n") {
					return fmt.Errorf("header values cannot contain line breaks")
				}
				target.Add(key, item.String)
			}
		default:
			return fmt.Errorf("header values must be strings or arrays of strings")
		}
	}
	return nil
}
func requestHTTP(a []Value) (Value, error) {
	if a[2].Kind != ObjectKind && a[2].Kind != NullKind {
		return Null(), fmt.Errorf("options must be an object or null")
	}
	options := a[2].Object
	for key := range options {
		switch key {
		case "body", "headers", "timeout_ms", "max_bytes", "follow_redirects":
		default:
			return Null(), fmt.Errorf("unknown request option %q", key)
		}
	}
	var body []byte
	var e error
	if value, ok := options["body"]; ok {
		body, e = valueBytes(value)
		if e != nil {
			return Null(), e
		}
	}
	timeout := 15 * time.Second
	if value, ok := options["timeout_ms"]; ok {
		timeout, e = timeoutDuration(value)
		if e != nil {
			return Null(), e
		}
	}
	maximum := int64(stdOutputLimit)
	if value, ok := options["max_bytes"]; ok {
		if value.Kind != IntegerKind || value.Integer < 1 || value.Integer > stdOutputLimit {
			return Null(), fmt.Errorf("max_bytes must be between 1 and 8388608")
		}
		maximum = value.Integer
	}
	follow := true
	if value, ok := options["follow_redirects"]; ok {
		if value.Kind != BooleanKind {
			return Null(), fmt.Errorf("follow_redirects must be boolean")
		}
		follow = value.Boolean
	}
	request, e := http.NewRequest(a[0].String, a[1].String, strings.NewReader(string(body)))
	if e != nil {
		return Null(), e
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" {
		return Null(), fmt.Errorf("URL must use http or https")
	}
	if value, ok := options["headers"]; ok {
		if e = applyHeaders(request.Header, value); e != nil {
			return Null(), e
		}
	}
	client := &http.Client{Timeout: timeout}
	if !follow {
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	response, e := client.Do(request)
	if e != nil {
		return Null(), e
	}
	defer response.Body.Close()
	data, e := io.ReadAll(io.LimitReader(response.Body, maximum+1))
	if e != nil {
		return Null(), e
	}
	if int64(len(data)) > maximum {
		return Null(), fmt.Errorf("HTTP response exceeds max_bytes")
	}
	return Object(map[string]Value{"status": Integer(int64(response.StatusCode)), "body": String(string(data)), "headers": headersValue(response.Header), "url": String(response.Request.URL.String())}), nil
}
