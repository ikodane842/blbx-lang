# BLBX

An interpreted language implemented in Go, with shared syntax checking for the
interpreter and command-line tools.

## Build and run

Install Go, then build from this directory:

```powershell
go build -o bin/blbx.exe .
.\bin\blbx.exe version
.\bin\blbx.exe check tests/first_test.bx
.\bin\blbx.exe run tests/first_test.bx
```

On macOS/Linux, build with `go build -o bin/blbx .` and use `./bin/blbx`.
For development, `go run . check tests/first_test.bx` also works. Use the built
executable when scripting exit codes; `go run` may wrap the program's exit status.

`run` executes the supplied file, sends program output to stdout, reads input
from stdin, and reports failures on stderr. It no longer writes AST, IR, or
runtime JSON into `tests/output`. Imports resolve using the interpreter's existing
module rules, rooted at the entry file's directory.

## Check source

```powershell
.\bin\blbx.exe check main.bx library.bx
.\bin\blbx.exe check --format json main.bx
'answer = )' | .\bin\blbx.exe check --format json --stdin-filename draft.bx -
```

Place options before file paths. Use `--` before a filename starting with a dash.
`-` reads source from stdin once. Explicit paths are checked in argument order;
directory traversal and glob expansion are not implemented.

Text mode prints diagnostics such as:

```text
draft.bx:1:10: error BX2002: expected expression, got ")"
```

Successful text checks are silent. JSON mode writes an array (or `[]` on success)
to stdout, including file read errors. This is the integration interface for a
future VS Code extension; no JavaScript parser or BLBX execution is involved.

```json
[
  {
    "file": "draft.bx",
    "line": 1,
    "column": 10,
    "endLine": 1,
    "endColumn": 11,
    "severity": "error",
    "code": "BX2002",
    "message": "expected expression, got \")\""
  }
]
```

Positions are **one-based Unicode code-point positions**, with exclusive end
positions. Tabs count as one column. Editor adapters must convert these positions
to their editor's indexing convention (VS Code uses zero-based UTF-16 positions).
EOF diagnostics have an empty range at the end of the source. Files must be
UTF-8; CRLF and an initial UTF-8 BOM are supported.

| Exit code | Meaning |
| --- | --- |
| `0` | Successful check or execution |
| `1` | Source diagnostics or runtime failure |
| `2` | Invalid command usage or file I/O failure |

`blbx help`, `blbx check --help`, and `blbx version` provide built-in help/version
information. Command usage errors go to stderr, including in JSON mode.

## Current checks and scope

| Code | Meaning |
| --- | --- |
| BX0001 | Cannot read source |
| BX1001 | Unknown character |
| BX1002 | Unterminated string |
| BX1003 | Unterminated block comment |
| BX1004 | Source is not UTF-8 |
| BX2001 | Missing or unexpected syntax token |
| BX2002 | Missing or unexpected expression |
| BX2003 | Invalid assignment target |
| BX2004 | Syntax nesting exceeds the safety limit |
| BX2005 | Invalid class body or duplicate constructor |

`check` is the first stage of the native BLBX linter: **lexical and syntax
validation**. It does not execute code, resolve imports, check undefined names,
infer types, validate function arity, or enforce style. A successful check does
not guarantee successful execution. Pass imported files explicitly to check them.

The parser discards a malformed statement and resumes on a later source line.
This allows multiple diagnostics, though malformed nested constructs can produce
follow-on errors. Recursive syntax nesting is limited to 256 parser levels.
Comments are accepted as trivia between tokens. Previously silently ignored
unknown characters and malformed syntax now produce errors.

The shared API is `syntax/check.Source(filename, source)`, which returns parsed
nodes and diagnostics. Callers must not execute/lower nodes when diagnostics are
present. `run` uses this API for entry files and imports and stops at the first
reported source error.

## Destructuring assignments

```text
[first, second, ...remaining] = [1, 2, 3, 4]
person = { "name": "Ada", "age": 36, "city": "London" }
{name, age: years, ...extra} = person
{city: home} = extra
[a, [b, c]] = [1, [2, 3]]
```

Array patterns bind by position; object patterns bind by stored field name.
`{name}` is shorthand for `{name: name}`. Quoted keys require an explicit target:
`{"display-name": label} = object`. Nested patterns work in either kind.
A final `...rest` captures remaining array elements or object fields into a new
shallow collection. Extra elements/fields are ignored when no rest binding exists.

The right-hand expression runs once. Missing elements/fields or mismatched types
raise runtime errors before any pattern bindings are written. Existing `null`
values bind normally. RHS side effects are not rolled back. Bindings follow the
existing variable-assignment scope rules. Extracted methods retain their receiver,
and object patterns can read stored fields from class instances.

Targets must be distinct variable names or nested patterns. Defaults, skipped
array slots, property/index targets, and parameter destructuring are not yet
supported. Use a separate line for a new array destructuring statement after an
expression, so it is distinguished from indexing that expression.

## Classes and self

```text
class Token {
    get_value = () => { return self.value }

    (_type, _value) => {
        self.type = _type
        self.value = _value
    }

    get_type = () => { return self.type }
}

tok = Token("STRING", "string")
print(tok)             // <Token instance>
print(tok.get_value()) // string
print(tok.get_type())  // STRING
```

The single unassigned anonymous function directly inside a class body is its
constructor. It can appear anywhere in the body. Named functions are methods;
other named assignments initialize fields. Each call creates a fresh instance,
initializes fields and methods in declaration order, then runs the constructor.
Mutable defaults are evaluated separately for each instance. The constructor's
return value is ignored. A class with no constructor accepts no arguments.
Multiple constructors and executable statements directly in a class body are
reported as errors.

Single inheritance uses `class Child extends Parent { ... }`. The parent must
already be a class; imported parents can use a dotted name such as `models.Base`.
Fields initialize from base to child, and child declarations override inherited
members. Mutable defaults remain separate per instance. Inherited methods keep
their defining module's lexical scope, while `self` refers to the child instance.

A child without a constructor inherits the nearest ancestor's constructor. A
child with its own constructor replaces it; parent constructors are not called
automatically. `super()` and multiple inheritance are not implemented.

```text
class Named {
    (name) => { self.name = name }
    label = () => { return self.name }
}
class Token extends Named {
    label = () => { return "Token: ".concat(self.name) }
}
print(Token("STRING").label()) // Token: STRING
```

`self` refers to the receiver and is available inside constructors, methods,
nested callbacks, and object-literal initializers. It cannot be reassigned or
used as a parameter; accessing it outside an object context is a runtime error.
Plain objects support it too:

```text
item = {}
item.value = 42
item.get = () => { return self.value }
print(item.get()) // 42
saved = item.get
print(saved())    // 42; extracted methods retain their receiver
```

Methods accessed through either dot or bracket notation bind `self`. Assigning a
method to a different object's property makes calls through that property use
the new receiver. Instances have type `object`; class definitions have type `class`.

## Type inspection

Plain objects expose `.indexes` and `.values` (also callable as `.indexes()` and
`.values()`). Keys are sorted, with values in the matching order. Both return
new shallow arrays; methods in `.values` retain their object receiver.

```text
data = { "b": 2, "a": 1 }
print(data.indexes.join(",")) // a,b
print(data.values.join(","))  // 1,2
```

These built-ins do **not** apply to class instances or class definitions.
Class instances may still define their own ordinary fields/methods with these
names. For plain objects with a field named `values` or `indexes`, bracket access
retrieves the stored field rather than the built-in property.

In BLBX, a **singleton** is a value enclosed in parentheses, such as `(11)` or
`("1")`, on which methods can be called. Parentheses preserve the underlying
type; they do not create a new runtime type. Method results can be chained.

```text
print((11).to_str())                // "11" (string)
print(("1").to_int())               // 1 (integer)
print(("1").to_float().typeof())    // float
print((true).not())                 // false
```

`to_str()` works on every value using its display representation. `to_int()` and
`to_float()` accept strings and numbers; `not()` is boolean-only. These methods
take no arguments. String-to-integer conversion accepts signed base-10 integers;
float-to-integer conversion truncates toward zero. Invalid conversions, integer
overflow, nonfinite string-to-float results, unsupported receivers, and extra
conversion arguments return `null`. String conversions do not trim whitespace.
Numeric arithmetic methods and string `concat()` retain their type-specific
behavior. Existing calls on variables and unparenthesized values remain supported.

Use `typeof(value)`, the singleton property `value.typeof`, or the method
`value.typeof()` to obtain a type
name as a string. Both support `null`, `boolean`, `integer`, `float`, `string`,
`array`, `object`, `function`, and `class`. Tuples currently evaluate to arrays.

```text
print(typeof(null))       // null
print((42).typeof)        // integer
print("hello".typeof)     // string
fn = () => {}
print(typeof(fn))         // function
```

`typeof` requires exactly one argument and evaluates it once. `.typeof` is a
built-in property even on objects; use bracket access for an object field named
`typeof`. The existing `.type()` method remains available.

## String and array singleton methods

Boolean singletons support `.and(other)`, `.or(other)`, and `.not()`:

```text
print((true).and(false))                  // false
print((false).or(true))                   // true
print((true).not())                      // false
print((2).lt(3).and((4).gt(1)))           // true
print((false).and(missing_function()))   // false; RHS is not evaluated
```

On booleans, `and` and `or` require one argument; `not` requires none. `and`
skips its argument when the receiver is false; `or` skips it when true.
An evaluated argument must be boolean. Wrong arity or a non-boolean evaluated
argument raises a runtime error. Method chains execute left to right. Ordinary
object methods named `and`, `or`, or `not` retain normal eager call behavior.
Non-boolean receivers without custom methods retain the existing `null` fallback.

Arrays support methods just like other singletons: `([1, 2]).join(",")`,
`items.join(",")`, and `[1, 2].join(",")` all work. Methods can be chained:

```text
print(("").concat("1"))                        // 1
print(("  red green  ").split().join(", "))   // red, green
print(("a,b,c").split(",").reverse().join("-")) // c-b-a
print(([1, 2]).append(3).join(","))             // 1,2,3
print(("-").join(["a", "b"]))                 // a-b
```

| Receiver | Method | Behavior |
| --- | --- | --- |
| String | `concat(values...)` | Append each argument's display text |
| String | `split()` | Split on whitespace, omitting empty parts |
| String | `split(separator)` | Split on an exact string, retaining empty parts; `""` splits into Unicode code points |
| String | `join(array)` | Use the receiver as the separator |
| String | `upper()`, `lower()` | Unicode case conversion |
| String | `trim()`, `trim_start()`, `trim_end()` | Remove whitespace from both ends, the start, or the end |
| String | `starts_with(text)`, `ends_with(text)` | Test a prefix or suffix |
| String | `replace(old, new)` | Replace the first occurrence |
| String | `replace_all(old, new)` | Replace every occurrence |
| String | `char_at(index)` | Return one code point; out of range returns `null` |
| String | `repeat(count)` | Repeat a nonnegative integer number of times; results over 16 MiB return `null` |
| Array | `join(separator = "")` | Join the elements' display text |
| Array | `concat(arrays...)` | Combine arrays, one level deep |
| Array | `append(values...)` | Return an array with the values appended |
| Array | `first()`, `last()` | Return the first/last element, or `null` for an empty array |
| Both | `length`, `length()` | String code-point count or array element count |
| Both | `is_empty()` | Test whether length is zero |
| Both | `contains(value)` | Test for a substring or matching array element |
| Both | `index_of(value)` | First matching position, or `-1` |
| Both | `slice(start = 0, end = length)` | Copy a range; end is exclusive |
| Both | `reverse()` | Return the reversed string/array |

These methods do not modify their receiver. Assign the result to keep it:
`items = items.append(3)`. Array copies are shallow; nested objects are shared.
Array searches use BLBX's existing equality rules (scalar value comparison,
including numeric cross-type equality; no deep array/object comparison).

String indices and lengths count Unicode code points, not UTF-8 bytes or visual
grapheme clusters. Negative `slice` and `char_at` indices count from the end;
slice bounds are clamped, and a reversed range returns an empty result. Until
negative number literals are supported, use `("-1").to_int()` for a negative
index. Invalid argument types/counts on the new methods return `null`.

## Undefined references

Using an undefined variable, calling an undefined function/method, reading a
missing object member (dot or string-key indexing), or calling a non-callable
value raises a runtime error. Member assignment requires an existing object:
use `obj = {}` before `obj.field = value`. New variables and new fields on
existing objects remain valid.

Explicit `null` values are valid and distinct from undefined names. Failed
conversions and out-of-range array reads still return `null`. Existing loop cursor
accessors such as `.elem()` remain supported. Errors currently stop execution
and make `blbx run` exit with code 1; catchable exceptions are not implemented.
These checks run during execution, not during the syntax-only `blbx check`.

## Standard library

For a complete BLBX project that exercises package imports, tokenization, and
parsing, see [the import CLI demo](import_cli/README.md). Run it with
`blbx run import_cli/main.bx`. Imported directories require `__init__.bx`;
all module bindings are accessible, including underscore-prefixed names.

Native modules are available as `std.strings`, `std.files`, `std.time`,
`std.networking`, `std.collections`, `std.serialization`, `std.math`, and
`std.processes`. Use `import std.math as math` or `from std.math import sqrt`.

See [the standard-library reference](docs/standard-library.md) for signatures,
return types, and limits. Run `blbx run examples/standard_library.bx` for a demo.

## Tests

```powershell
go test ./...
go vet ./...
go test ./syntax/check -run=^$ -fuzz=FuzzSource -fuzztime=15s
```

Tests cover valid language constructs, invalid syntax, diagnostic positions,
parser reuse, CLI output/exit codes, stdin, imports, and existing interpreter
behavior. If your environment restricts the default Go build cache, set
`GOCACHE` to a writable directory before building.
