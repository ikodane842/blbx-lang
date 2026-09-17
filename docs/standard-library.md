# BLBX standard library

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

This is the initial standard-library API. Arguments below are required, with
exact arity. Bad types, failed I/O, and unsupported numeric domains raise runtime
errors and stop execution; they do not silently return `null`. Functions are
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
There is no server, socket, streaming, or custom-header API yet.

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

## Serialization — `std.serialization`

| Function | Result |
| --- | --- |
| `json_encode(value)` | Compact JSON string |
| `json_decode(text)` | BLBX value |

JSON supports null, booleans, strings, integers, floats, arrays, and objects.
Integers preserve signed 64-bit precision. Out-of-range JSON integers, trailing
content, malformed input, nonfinite floats, functions, and class definitions are
errors. Cycles or nesting deeper than 128 levels are rejected. Objects containing
methods cannot be serialized directly; construct a data-only object first.
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
