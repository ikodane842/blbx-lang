# Files, OS, JSON, networking, and security

[Documentation index](README.md) · [Standard library](standard-library.md) · [Tasks](async.md)

All functions below are included in the executable. Use `import std.files as files`,
`import std.os as os`, `import std.serialization as json`,
`import std.networking as net`, and `import std.security as security`.
Arguments are required unless an option field is explicitly described as optional.
Failures use the standard diagnostic format and can be caught with `try`.

## Files and paths — std.files

Paths used for filesystem operations resolve beside the source file that imported
`std.files`. Absolute paths are unchanged. Pure path transformations do not touch
the filesystem. Native separators and path rules follow the host OS.

| Function | Return value / behavior |
| --- | --- |
| `read(path)` | Entire file as a string; existing text-read API |
| `write(path, text)` | Create/overwrite text; null |
| `append(path, text)` | Create/append text; null |
| `read_bytes(path)` | Array of integer bytes; maximum 1 MiB |
| `write_bytes(path, bytes)` | Create/overwrite bytes; null |
| `exists(path)` | Whether a path exists; other stat failures raise errors |
| `is_file(path)` | Whether a regular file exists; follows symlinks |
| `is_dir(path)` | Whether a directory exists; follows symlinks |
| `stat(path)` | Object described below; follows symlinks |
| `mkdir(path)` | Create directory and missing parents; null |
| `list(path)` | Sorted immediate entry names |
| `walk(path)` | Absolute paths, including the root, in lexical traversal order |
| `copy(source, destination)` | Bytes copied; destination must not already exist |
| `rename(source, destination)` | Move/rename using host OS semantics; null |
| `remove(path)` | Remove one file, symlink, or empty directory; null |
| `join(first, second)` | Join two path strings |
| `basename(path)` | Final component |
| `dirname(path)` | Parent component |
| `extension(path)` | Extension including dot, or empty string |
| `clean(path)` | Normalize redundant separators and dot components |
| `is_absolute(path)` | Boolean |
| `absolute(path)` | Absolute path using the importing file's base directory |
| `relative(base, target)` | Relative path from base to target; both resolve against the importer |

`stat` returns `name`, `size` in bytes, `is_dir`, `is_file`, `modified_ms` as Unix
milliseconds, and `mode` as permission bits. Windows permission semantics differ
from Unix; mode is not an ACL description.

`copy` accepts regular files and streams them without an 8 MiB size limit. It
copies permission bits, not timestamps or other metadata. Failed copies remove
the newly created destination. `rename` may overwrite an existing destination
where the host OS permits it; cross-filesystem moves can fail. `walk` does not
follow symlinks and stops with an error above 100,000 entries. `remove` is never
recursive and does not follow a symlink to remove its target.

Byte arrays contain integers 0–255. Byte-input operations accept up to 8 MiB;
`read_bytes` limits its returned array to 1 MiB. Whole-text `read` retains its
existing behavior and does not impose the JSON/binary read limit. Writes require
existing parent directories. Errors use BX4101.

## Operating system — std.os

| Function / constant | Return value / behavior |
| --- | --- |
| `platform` | Host OS name, such as `windows`, `linux`, or `darwin` |
| `arch` | CPU architecture, such as `amd64` or `arm64` |
| `cwd()` | Process working directory |
| `home()` | Current user's home directory |
| `temp_dir()` | Host temporary-directory path |
| `env(name)` | Environment value or null when unset |
| `environment()` | Snapshot object of environment variables |
| `set_env(name, value)` | Set process environment; null |
| `unset_env(name)` | Remove process environment entry; null |
| `args()` | Script arguments, excluding executable and script path |
| `read_line()` | One line without its newline, or null at EOF |
| `write_stdout(text)` | Write exactly the supplied string; null |
| `write_stderr(text)` | Write exactly the supplied string; null |
| `run(executable, arguments)` | Process result with `exit_code`, `stdout`, and `stderr` |

```powershell
.\bin\blbx.exe run app.bx first second
```

Within that script, `os.args()` is `["first", "second"]`. Arguments beginning with
`-` after the script path are script arguments. To include a literal `--`, pass it
as an argument; it is not stripped. The existing `std.processes` API remains
available, and `os.run` has the same direct-execution, 15-second timeout, and
8 MiB-per-output-stream limits. Nonzero child exit codes are returned normally.
There is no implicit shell or child stdin API.

`read_line` distinguishes an empty line (`""`) from EOF (`null`) and limits a line
to 8 MiB. The write functions do not append a newline. In tasks, stdout and stderr
are buffered separately and replayed once to their corresponding streams at await;
relative interleaving between those two streams is not preserved. Script arguments
are copied into workers. Worker stdin is empty.

Environment changes are process-wide and visible across tasks and later child
processes; they are not isolated BLBX variables. Changing the environment does not
change the invoking terminal's environment. Errors use BX4109. There is no `chdir`
API: file imports retain their explicit path base, and concurrent tasks share one
process working directory.

## JSON — std.serialization

| Function | Return value / behavior |
| --- | --- |
| `json_encode(value)` | Compact JSON text |
| `json_decode(text)` | Decoded BLBX value |
| `json_valid(text)` | Whether text can be decoded within BLBX's JSON rules and limits |
| `json_pretty(value, indent)` | JSON text with 0–8 indentation spaces |
| `json_read(path)` | Read and decode a JSON file |
| `json_write(path, value)` | Encode and overwrite a JSON file; null |

File paths resolve beside the file importing `std.serialization`. Writes require
an existing parent directory. Invalid input is an error except in `json_valid`,
which returns false. Input/output text is limited to 8 MiB; encoding also applies
an in-memory size budget. Decoding allows at most 100,000 JSON tokens and 128
nested containers. UTF-8 is validated. Signed 64-bit integers preserve their
precision; integer overflow is rejected. Decimal/exponent numbers become floats.
Duplicate keys use the last value, matching Go JSON decoding. Malformed input,
trailing values, nonfinite numbers, cycles, and excessive nesting are rejected.

Arrays and tuples encode as JSON arrays, which decode into BLBX arrays. Objects
and class instances encode their stored fields; class identity is not preserved.
Functions, classes, interfaces, tasks, and network resources cannot be encoded.
Object fields containing these values also fail. Errors use BX4105.

```text
import std.serialization as json
value = json.json_decode("{\"name\":\"BLBX\",\"ready\":true}")
print(json.json_pretty(value, 2))
print(json.json_valid("not JSON")) // false
```

## HTTP client — std.networking

Existing `get(url)` and `post(url, body, content_type)` still work. For options:

```text
import std.networking as net
response = net.request("GET", "https://example.com", {
    "headers": {"Accept": "text/plain"},
    "timeout_ms": 5000,
    "max_bytes": 1048576,
    "follow_redirects": true
})
print(response.status)
print(response.body)
```

`request(method, url, options)` accepts an options object or null for defaults.
Because an empty block argument evaluates to null, passing `{}` also selects the
defaults. Unknown option names are errors.

| Option | Type and default |
| --- | --- |
| `body` | String or byte array; default empty |
| `headers` | Object of strings or arrays of strings; default none |
| `timeout_ms` | Integer 1–300,000; default 15,000 |
| `max_bytes` | Maximum response-body bytes, 1–8,388,608; default 8 MiB |
| `follow_redirects` | Boolean; default true |

The response is `{status, body, headers, url}`. `body` is a string containing the
response bytes; headers use lowercase keys and arrays of strings. `url` is the
final URL. Existing get/post responses retain their earlier fields without url.
Only HTTP and HTTPS are accepted. HTTPS uses Go's normal certificate verification.
Disabling redirects returns the redirect response; default redirect behavior
follows the Go client. Non-2xx statuses are responses, not exceptions. Header
names and values are validated; CR/LF injection is rejected. Errors use BX4103.
Request body input is limited to 8 MiB. This API does not expose streaming,
websockets, custom CA configuration, or TLS-verification bypasses.

## TCP, UDP, DNS, and URL helpers — std.networking

| Function | Return value / behavior |
| --- | --- |
| `tcp_connect(address, timeout_ms)` | Connected TCP resource |
| `tcp_listen(address)` | TCP listener resource |
| `tcp_accept(listener, timeout_ms)` | Accepted TCP connection resource |
| `tcp_send(connection, data, timeout_ms)` | Bytes written; data is string or byte array |
| `tcp_receive(connection, max_bytes, timeout_ms)` | `{data: byte_array, eof: boolean}` |
| `udp_bind(address)` | UDP resource |
| `udp_send(socket, address, data, timeout_ms)` | Bytes sent |
| `udp_receive(socket, max_bytes, timeout_ms)` | `{data: byte_array, address: sender_string}` |
| `address(resource)` | Local bound address, including the actual port |
| `close(resource)` | Close listener, socket, or HTTP server; null |
| `resolve_host(host, timeout_ms)` | Array of IP address strings |
| `url_encode(text)` | URL query-component encoding |
| `url_decode(text)` | URL query-component decoding; plus signs become spaces |

Addresses use `host:port`; bracket IPv6 addresses. Port 0 requests an available
local port when binding. All timeout arguments are required integers from 1 to
300,000 milliseconds. Timeouts begin after a call obtains its resource's
per-direction I/O lock; time queued behind another operation is not included.
DNS for UDP sends is included in that send's timeout. Listener/bind hostname
resolution has no explicit timeout argument.

Receives accept `max_bytes` from 1 to 1,048,576. TCP is a byte stream: one receive
may return fewer bytes than requested, and a send is not a message boundary.
Loop until your application framing rule or EOF is reached. EOF may accompany
data; process data before stopping. A failed send may already have written a
prefix. UDP retains datagram boundaries, limits sends to 65,507 bytes, and raises
an error if a received datagram exceeds max_bytes rather than silently returning
a truncated message; that datagram has been consumed. Delivery is not guaranteed.

Resources have type `resource`, cannot be forged from BLBX objects, and cannot be
JSON-serialized. Unlike ordinary values, they are shared across task snapshots:
closing one handle closes it for all callers. Close is repeatable. Closing from
another task interrupts pending I/O. Concurrent reads or writes on one resource
serialize per direction. Prefer one reader and one writer per connection.
Resources need explicit close; they do not keep the CLI alive after script exit.
The current socket API provides TCP/UDP, not TLS sockets or Unix-domain sockets.

## HTTP server — std.networking

`serve(address, handler)` starts a server and returns an HTTP-server resource.
It binds immediately; use `address(handle)` to discover an ephemeral port.
`wait(handle)` blocks until the server stops; `close(handle)` stops listening and
closes active connections. Close is immediate, not a graceful handler drain.

```text
import std.networking as net
server = net.serve("127.0.0.1:0", (request) => {
    return {"status": 200, "body": "Hello, BLBX"}
})
response = net.get("http://".concat(net.address(server)))
print(response.body)
net.close(server)
net.wait(server)
```

For a long-running service, call `wait` after starting it. A handler receives:

| Request field | Type |
| --- | --- |
| `method`, `path`, `url` | Strings; url is the request target, usually relative |
| `query` | Object mapping original query keys to arrays of strings |
| `headers` | Object mapping lowercase header names to arrays of strings |
| `body` | Raw request bytes represented as a string |
| `remote_address` | Client address string |

Return a string or byte array for status 200, or return an object with optional
`status` (integer 200–599), `headers`, and `body` fields. The default body is empty.
Unknown response fields and invalid values produce a 500 response. Uncaught
handler errors also produce a generic 500; their diagnostic details are retained
in server output rather than disclosed to the client. Use `try` inside a handler
to choose an application-specific error response.

Every request runs against a fresh snapshot of the handler's state captured at
serve time. Mutations do not persist to the next request or the caller. Sockets,
files, and other external effects are still shared resources. Each server admits
at most 64 simultaneous handlers; excess requests receive 503. Incoming bodies
and outgoing response bodies are limited to 8 MiB. Header-read timeout is 5 seconds;
read/write timeouts are 15 seconds; idle timeout is 60 seconds; headers are capped
at 1 MiB. Socket timeouts and close do not cancel BLBX code already executing.
A handler with an infinite loop can continue running until process exit.

`server_output(handle)` returns `{stdout, stderr, truncated}` and drains captured
handler output. Each stream retains at most 1 MiB between drains. Output is added
when a handler finishes, not streamed live. Handler runtime errors appear in
stderr; buffer truncation sets the flag. Handler stdin is empty. There is no TLS
listener, routing framework, streaming response, or websocket API yet; dispatch
on `request.method` and `request.path` in BLBX.

## Security — std.security

These functions use Go's established crypto and encoding implementations.
`data` and `key` accept a string's raw bytes or an array of integers 0–255,
with an 8 MiB input limit. There are no implicit hex/base64 conversions.

| Function | Return value / behavior |
| --- | --- |
| `random_bytes(count)` | Cryptographically secure byte array; 0–1,048,576 bytes |
| `random_hex(count)` | Secure random bytes encoded as lowercase hex; same byte-count limit |
| `random_int(minimum, maximum)` | Uniform secure integer in `[minimum, maximum)`; minimum must be smaller |
| `md5(data)` | 32-character lowercase hexadecimal MD5 digest |
| `sha256(data)` | 64-character lowercase hexadecimal SHA-256 digest |
| `sha512(data)` | Lowercase hexadecimal SHA-512 digest |
| `hmac_sha256(key, data)` | Lowercase hexadecimal HMAC-SHA-256 tag |
| `verify_hmac_sha256(key, data, expected_hex)` | Constant-time tag comparison; malformed tag is an error |
| `constant_time_equal(first, second)` | Byte comparison without content-dependent early exit; length is observable |
| `hex_encode(data)` | Lowercase hexadecimal text |
| `hex_decode(text)` | Byte array; invalid hex is an error |
| `base64_encode(data)` | Standard padded base64 text |
| `base64_decode(text)` | Byte array using strict standard base64 decoding |
| `utf8_encode(text)` | UTF-8 byte array; validates text |
| `utf8_decode(bytes)` | String; invalid UTF-8 is an error |

Decoding and utf8_encode return at most 1 MiB of bytes. Hex/base64 encoded input
is capped at 2 MiB; decoded size is checked separately. Base64 uses the standard
alphabet with padding, not the URL-safe alphabet. Go's decoder accepts CR/LF in
base64 input but rejects invalid alphabet/padding bits. HMAC tags must be exactly
64 hex characters. Errors use BX4110. Raw hashes are not password-hashing schemes;
this API does not yet provide password KDFs, encryption, or certificate management.

```text
import std.security as security
key = security.random_bytes(32)
tag = security.hmac_sha256(key, "message")
print(security.verify_hmac_sha256(key, "message", tag)) // true
print(security.md5("abc")) // 900150983cd24fb0d6963f7d28e17f72
print(security.sha256("abc")) // ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
print(security.md5([97, 98, 99])) // same digest as "abc"
```

## Examples and diagnostic categories

Run `examples/networking.bx` for a local HTTP round trip, and
`examples/security_json.bx` for JSON and security primitives. All failures are
catchable with `try(work, handler)`. Standard arity/type validation uses BX4003
and BX4004; file, JSON, and network operations use BX4101, BX4105, and BX4103.
OS and security failures use BX4109 and BX4110.
