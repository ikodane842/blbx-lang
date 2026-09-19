# BLBX standard library

[Documentation index](README.md) · [Language](language.md) · [Async guide](async.md)

Native modules ship inside the BLBX executable. Importing them does not perform
I/O. `check` validates import syntax without loading modules or executing calls.
Bundled module names take precedence over files with the same dotted path.

```text
import std.files as files
import std.serialization as json
from std.math import sqrt

print(sqrt(81))
data = json.json_decode("{\"name\":\"BLBX\"}")
print(data.name)
files.write("data.json", json.json_encode(data))
```

Except for the variadic `task.spawn`, arguments below are required, with
exact arity. Bad types, failed I/O, and unsupported numeric domains raise runtime
errors and stop execution unless caught with `try`. Documented absence cases, such as an unset environment
variable, may return `null`. Functions are
ordinary callable values, so aliases and `from ... import ...` work.

## Strings — `std.strings`

| Function | Result |
| --- | --- |
| `concat(text, suffix)` | Combined string; both arguments must be strings |
| `split(text, separator)` | Array; an empty separator splits into Unicode code points |
| `join(array, separator)` | String using each element's display text |
| `trim(text)`, `trim_start(text)`, `trim_end(text)` | Whitespace removal |
| `upper(text)`, `lower(text)`, `reverse(text)` | Transformed string |
| `contains(text, part)`, `starts_with(text, part)`, `ends_with(text, part)` | Boolean |
| `index_of(text, part)` | Code-point index, or `-1` |
| `replace(text, old, new)` | Replace **all** occurrences |

These supplement singleton methods. Note that `std.strings.replace` replaces all
matches, whereas the singleton `.replace()` replaces the first match.

## Files — `std.files`

The additional byte, stat, copy, rename, remove, walk, and path APIs are listed in
[the complete files/OS reference](system-library.md#files-and-paths--stdfiles).

| Function | Result |
| --- | --- |
| `read(path)` | File contents as a string |
| `write(path, text)` | Create or overwrite a file; returns `null` |
| `append(path, text)` | Create or append; returns `null` |
| `exists(path)` | Boolean; other stat errors are reported |
| `mkdir(path)` | Create directory and missing parents; returns `null` |
| `list(path)` | Sorted array of immediate entry names |
| `join(first, second)` | Platform-specific joined path |

Relative paths resolve beside the `.bx` file that imports `std.files`, including
when a function imported from that module is called later. Absolute paths are
used unchanged. For example, `tests/main.bx` reading `"hello.txt"` reads
`tests/hello.txt`, even when launched from the project root.
File functions read/write the whole string; streaming and binary buffers are
not yet exposed. `write` and `append` do not create missing parent directories.

## Time — `std.time`

| Function | Result |
| --- | --- |
| `now()` | Integer Unix timestamp in **milliseconds** |
| `sleep(milliseconds)` | Block execution; accepts 0–86,400,000; returns `null` |
| `format(milliseconds)` | UTC RFC 3339 timestamp string |
| `parse(timestamp)` | RFC 3339 string to Unix milliseconds |

## Networking — `std.networking`

| Function | Result |
| --- | --- |
| `get(url)` | HTTP response object |
| `post(url, body, content_type)` | HTTP response object; all arguments are strings |

Responses contain `status` (integer), `body` (string), and `headers` (object).
Header keys are lowercase; each header value is an array of strings. HTTP error
statuses such as 404 remain ordinary responses; transport failures are errors.
Only HTTP/HTTPS URLs are supported. Requests follow the Go HTTP client's normal
redirect behavior, time out after 15 seconds, and limit response bodies to 8 MiB.
Configurable requests, TCP/UDP, DNS, and an HTTP server are now available; see
[the networking reference](system-library.md#http-client--stdnetworking).

## Collections — `std.collections`

| Function | Result |
| --- | --- |
| `length(value)` | Array length, string code-point count, or object field count |
| `keys(object)` | Sorted array of field names |
| `values(object)` | Values in the same sorted-key order |
| `range(count)` | Array of integers from 0 up to but excluding count; 0–1,000,000 |
| `contains(array, value)` | Boolean using existing BLBX equality rules |

Array singleton methods provide joining, slicing, concatenation, appending, and
reversal. These transformations return new shallow arrays.
`length` takes one array, string, or object; `keys` and `values` take an object;
`range` takes an integer; `contains` takes an array and any value.
The module functions accept class instances as objects. This differs from the
plain-object-only `.indexes`/`.values` singleton accessors. Module `values`
returns stored values directly, without the receiver binding performed by those
singleton accessors; use `instance.method` to extract a bound method.

## Serialization — `std.serialization`

Also available: `json_valid(text)`, `json_pretty(value, indent)`, `json_read(path)`,
and `json_write(path, value)`. See [JSON helpers and limits](system-library.md#json--stdserialization).

| Function | Result |
| --- | --- |
| `json_encode(value)` | Compact JSON string |
| `json_decode(text)` | BLBX value |

JSON supports null, booleans, strings, integers, floats, arrays, tuples, and objects.
Tuples encode as JSON arrays and decode back as BLBX arrays.
Integers preserve signed 64-bit precision. Out-of-range JSON integers, trailing
content, malformed input, nonfinite floats, functions, task handles, and class definitions are
errors. Cycles or nesting deeper than 128 levels are rejected. Class instances serialize their stored fields without preserving class identity. Objects containing
methods cannot be serialized directly; construct a data-only object first.
JSON input/output is limited to 8 MiB, decoding to 100,000 tokens, and encoding
to an additional value-size budget. Input strings must be valid UTF-8.
Exponent/decimal JSON numbers decode as floats. Other formats are not yet exposed.

## Math — `std.math`

Constants: `pi`, `e`.

Functions: `abs(x)`, `sqrt(x)`, `floor(x)`, `ceil(x)`, `round(x)`, `sin(x)`,
`cos(x)`, `log(x)`, `pow(x, y)`, `min(x, y)`, `max(x, y)`.

Inputs may be integers or floats; results are floats. Calculations use float64,
so very large integers may lose precision. `log` is the natural logarithm, angles
are radians, and `round` rounds halfway values away from zero. Nonfinite results
are errors.

## Processes — `std.processes`

| Function | Result |
| --- | --- |
| `run(executable, arguments)` | Object with `exit_code`, `stdout`, and `stderr` |
| `env(name)` | Environment-variable string, or `null` if unset |
| `cwd()` | Current working directory |

`arguments` must be an array of strings. Execution passes arguments directly;
there is no implicit shell, expansion, or pipeline. The child inherits the
working directory and environment, and receives no stdin. Nonzero process exits
are returned in `exit_code`; launch failures are errors. Calls time out after
15 seconds, with up to 8 MiB each of stdout and stderr. Timeout stops the direct
child; this API does not manage descendant process trees or background jobs.

## Asynchronous tasks — `std.task`

```text
import std.task as task
import std.time as time

work = (value) => {
    time.sleep(100)
    return value.mul(2)
}
first = task.spawn(work, 21)
second = task.spawn(work, 10)
print(task.await(first))   // 42
print(task.await(second))  // 20
```

- `spawn(function, ...arguments)` starts work concurrently and returns a `task` handle.
- `await(handle)` waits, returns a copy of the result, and propagates worker errors as runtime errors.
- `done(handle)` returns whether the worker has finished, including failure.

Each worker receives a snapshot of captured variables and arguments. Objects,
arrays, closures, and class instances are isolated from the caller, with cycles
and aliases preserved within the snapshot. External effects such as file writes
and network requests still affect the same external resources. Repeated awaits
return independent snapshots. Nested tasks and returning closures are supported.

Worker output is buffered and emitted once, by the first await. Workers have
empty standard input. Await every task you need before the main program exits;
unawaited tasks do not keep the CLI alive. There is currently no cancellation,
worker pool, or task timeout. Task failures are raised when awaited and can be caught with `try`, like
other runtime failures.

## OS and security modules

See [std.os](system-library.md#operating-system--stdos) and
[std.security](system-library.md#security--stdsecurity) for every signature and limit.

## More examples

String transformations (no I/O):

```text
import std.strings as strings
print(strings.replace("a-a", "a", "b")) // b-b
print(strings.join(strings.split("red,green", ","), " / "))
```

Working with time and JSON (no external I/O):

```text
import std.time as time
import std.serialization as json
print(time.format(time.parse("1970-01-01T00:00:00Z")))
text = json.json_encode({"ready": true, "items": [1, 2]})
print(json.json_decode(text).ready)
```

File round trip (creates `data/message.txt` beside this script):

```text
import std.files as files
files.mkdir("data")
path = files.join("data", "message.txt")
files.write(path, "Hello")
files.append(path, " BLBX")
print(files.read(path))
print(files.list("data"))
```

HTTP GET (requires network access):

```text
import std.networking as network
response = network.get("https://example.com")
print(response.status)
print(response.body)
```

Child process (requires the `blbx` executable on PATH):

```text
import std.processes as process
result = process.run("blbx", ["version"])
print(result.exit_code)
print(result.stdout)
```

## Scope of the library

The [system-library reference](system-library.md) documents expanded file, OS,
JSON, network, and security exports. `std.os` supplies environment setters,
script arguments, streams, and process execution. `std.security` supplies secure
randomness, hashes, HMAC, and encoding utilities. Regular expressions, date
arithmetic, a standard test framework, and a package manager remain unavailable. Use the supported signatures rather than assuming APIs from Python,
JavaScript, or Go are present.

For state isolation, output ordering, failures, polling, nested jobs, and batching,
see the [complete async guide](async.md).
