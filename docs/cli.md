# BLBX command-line reference

[Documentation index](README.md) � [Language](language.md)

## Build and run

Install Go, then build from this directory:

```powershell
go build -o bin/blbx.exe .
.\bin\blbx.exe version
.\bin\blbx.exe check examples/standard_library.bx
.\bin\blbx.exe run examples/standard_library.bx
```

On macOS/Linux, build with `go build -o bin/blbx .` and use `./bin/blbx`.
For development, `go run . check examples/standard_library.bx` also works. Use the built
executable when scripting exit codes; `go run` may wrap the program's exit status.

`run` executes the supplied file, sends program output to stdout, reads input
from stdin, and reports failures on stderr. Imports resolve using the interpreter's existing
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
syntax BX2006: unmatched closing delimiter ")"
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
    "code": "BX2006",
    "message": "unmatched closing delimiter \")\""
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
| BX2005 | Invalid class body |
| BX2006 | Unmatched or misplaced delimiter |
| BX2007 | Missing comma |
| BX2008 | Invalid object field syntax |
| BX2009–BX2013 | Self binding, destructuring, and duplicate constructor errors |
| BX3001 | Undefined variable or function name |
| BX3002 | Undefined object member |
| BX4001 | Unclassified native operation failure |
| BX4002–BX4013 | Specific runtime validation, import, input, and task failures |
| BX4101–BX4108 | Standard-library operation failures, separated by module |

See [the complete error-code reference](error-codes.md) for each code's
individual meaning. All emitted codes come from one registry in
`syntax/diagnostic/codes.go`. Each category has a distinct code; repeated errors
in the same category reuse it. Wrapping an error adds context without changing
its underlying code.

Text diagnostics use `syntax CODE: message` or `runtime CODE: message`, for
example `syntax BX3001: undefined variable "start"`. Runtime undefined names
use the same BX3001 code with the runtime prefix. Standard-library and task
errors use the same format. File and position metadata remain available in
`check --format json`; compact text diagnostics omit source locations.

Commas are required between function arguments (including `if` branches), array
elements, and object fields. Newlines do not replace commas. Trailing commas are
allowed; leading commas and empty entries are rejected. Function parameters also
require commas. Ordinary function and class bodies remain statement blocks.

```text
if(true, () => { print("yes") }, () => { print("no") })
items = [1, 2, 3]
object = {"name": "BLBX", "ready": true}
```

`check` performs lexical, syntax, and conservative undefined-name validation.
It checks inside functions and branches even when they never execute. `run`
performs the same checks before execution. Missing commas can be reported
alongside undefined names; other syntax failures may prevent name checking.

Name checking recognizes parameters, assignments, destructuring, classes, and
import bindings. It allows forward references and recursion. This is not
definite-assignment analysis: a name assigned elsewhere may still be unavailable
when execution reaches a read. Dynamic members and names potentially inherited
from imported or dynamically selected classes remain runtime checks.

The checker does not execute code, resolve imports, infer types, validate
function arity, or enforce style. A successful check does
not guarantee successful execution. Pass imported files explicitly to check them.

The parser discards a malformed statement and resumes on a later source line.
This allows multiple diagnostics, though malformed nested constructs can produce
follow-on errors. Recursive syntax nesting is limited to 256 parser levels.
Comments are accepted as trivia between tokens. Previously silently ignored
unknown characters and malformed syntax now produce errors.

The shared API is `syntax/check.Source(filename, source)`, which returns parsed
nodes and diagnostics. Callers must not execute/lower nodes when diagnostics are
present. `run` uses this API for entry files and imports and refuses execution when diagnostics are present. Entry-file diagnostics
are reported together; imported-file checking currently returns its first error.


## Build validation and releases

```powershell
go vet ./...
go build -trimpath -ldflags "-s -w" -o bin/blbx.exe .
.\bin\blbx.exe check examples/standard_library.bx
.\bin\blbx.exe run examples/standard_library.bx
```

Automated tests and the tests fixture directory have been removed. Runnable
examples remain. Build/vet and manual smoke checks do not replace regression tests.

The CLI supports `run`, `check`, `help`, and `version`. BLBX is still an alpha:
variable scope rules need refinement, indexed assignment is incomplete,
user-function arity is permissive, and runtime errors cannot yet be caught.
The checker validates syntax and conservatively checks undefined names. There is no general REPL or editor extension.

The default version is `0.1.0-dev`. Set a version for a release build, for example:

```powershell
go build -trimpath -ldflags "-s -w -X blbx_lang/internal/cli.Version=0.1.0-alpha.1" -o bin/blbx.exe .
```

Go is required to build, but not to run the resulting executable. Include LICENSE
and access to the matching source when distributing a release. If the default
Go cache is restricted, set `GOCACHE` to a writable directory before building.
