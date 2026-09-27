# BLBX

BLBX is a dynamically typed, interpreted language implemented in Go. It supports
first-class functions, callback-based control flow, objects, classes, imports,
type-specific singleton methods, and asynchronous tasks.

Two execution backends share the same syntax and frontend: the original
interpreter (`blbx run`) and Black Box's graph runtime (`blackbox run` or
`blbx run-graph`).

Recent additions include formatted strings (`f"Hello, {name}"`), integer
bitwise methods, Unicode `ord()`, object-key membership (`key.in(object)`),
and in-place array `extend()`. `while` callbacks now receive a loop cursor.
Returns inside `if`, `for`, and `while` callbacks exit the enclosing ordinary
function; `assert(value)` exits the nearest executing control-flow statement.
See the [language reference](docs/language.md) and
[focused examples](docs/README.md#focused-examples).

```text
import std.task as task

double = (value) => { return value.mul(2) }
job = task.spawn(double, 21)
print(task.await(job)) // 42
```

## Build and run

Run these commands from the repository root with Go installed:

```powershell
go build -trimpath -ldflags "-s -w" -o bin/blbx.exe .
.\bin\blbx.exe check examples/language_tour.bx
.\bin\blbx.exe run examples/language_tour.bx
```

On macOS/Linux, build with `go build -o bin/blbx .`, then use `./bin/blbx`.
Go is required to build, but not to run the executable.
Build output belongs in the ignored `bin/` directory. These commands rebuild
`bin/blbx.exe`; they do not replace the repository's root `blbx.exe`.

## Black Box graph execution

Black Box uses the same BLBX syntax and IR, then builds and traverses an execution
graph. Explicit calls execute even when their return values are discarded;
branches and boolean short-circuiting determine which paths are needed.

The backend separates **declarative scalar value graphs** from **ordered effects**.
Proven pure calculations can share a result within an activation. Binding versions
capture values at assignment time; calls, mutation, allocation, and I/O keep their
execution order. Unknown or user-defined calls are never assumed pure. See
[declarative_values.bx](examples/declarative_values.bx) for runnable examples.

Literals are static leaf nodes. Control-flow regions select branches and traverse
loop gates, while effect dependencies preserve execution order. Assignments
capture values rather than reactive formulas; execution is not universally lazy.

```powershell
go build -trimpath -ldflags "-s -w" -o bin/blackbox.exe ./cmd/blackbox
.\bin\blackbox.exe run examples/declarative_values.bx
.\bin\blackbox.exe graph examples/declarative_values.bx > bin/graph.json
```

The regular CLI also exposes `blbx run-graph file.bx` and `blbx graph file.bx`.
`blbx run` retains the original interpreter for comparison. On macOS/Linux,
build with `go build -o bin/blackbox ./cmd/blackbox` and use `./bin/blackbox`.
See [graph execution and object representation](docs/graph-runtime-design.md).
The `graph` command exports JSON without executing the script.

## Run blbx from anywhere on Windows

Place `blbx.exe` in a permanent folder, such as `%LOCALAPPDATA%\Programs\BLBX`.
Search Start for **Edit environment variables for your account**, edit **Path**
under **User variables**, and add that folder as a new entry. Add the folder,
not the executable filename, and keep the existing entries.

Restart your terminal (and VS Code if using its terminal), then verify:

```powershell
blbx version
blbx help
blbx run "C:\path\to\main.bx"
```

See [Windows installation and PATH setup](docs/cli.md#windows-installation-and-path-setup)
for step-by-step instructions and troubleshooting. No administrator access or Go
installation is needed to run a prebuilt Windows executable.

## Documentation

VS Code syntax highlighting for `.bx` files is available in
[editors/vscode](editors/vscode/README.md), with installation and development instructions.
Generate the VSIX package from that directory using `npm run package`; generated
packages are ignored by Git. The extension provides highlighting and bracket
support, without a language server or editor diagnostics.

- [Complete documentation index](docs/README.md)
- [Language syntax and singleton methods](docs/language.md)
- [Exceptions, interfaces, operators, tuples, and inheritance](docs/features.md)
- [Standard library](docs/standard-library.md)
- [Files, OS, JSON, networking, and security](docs/system-library.md)
- [Async and await usage](docs/async.md)
- [CLI, diagnostics, and release builds](docs/cli.md)
- [Graph runtime, values, effects, and objects](docs/graph-runtime-design.md)
- [Error codes](docs/error-codes.md)

## Development checks

After building both binaries, check the Go packages and run a focused example:

```powershell
go vet ./...
.\bin\blbx.exe check examples/declarative_values.bx
.\bin\blbx.exe run examples/declarative_values.bx
.\bin\blackbox.exe run examples/declarative_values.bx
```

Runnable examples live in `examples/`. There is no automated test suite or test
fixture directory; build/vet and example checks do not replace regression tests.
Keep temporary verification scripts and generated graph exports out of source
directories. See the [CLI guide](docs/cli.md#build-validation-and-releases) for
release versioning.

## Status

The current default version is `0.1.0-dev`. BLBX is an alpha: variable scope,
user-function parameter validation, and numeric overflow handling have known limits.
Runtime failures can be caught with callback-style `try`. The native checker validates syntax and
conservatively detects undefined names before execution, including inside
uncalled branches and functions.
