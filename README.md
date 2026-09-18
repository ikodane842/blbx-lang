# BLBX

BLBX is a dynamically typed, interpreted language implemented in Go. It supports
first-class functions, callback-based control flow, objects, classes, imports,
type-specific singleton methods, and asynchronous tasks.

```text
import std.task as task

double = (value) => { return value.mul(2) }
job = task.spawn(double, 21)
print(task.await(job)) // 42
```

## Build and run

```powershell
go build -o bin/blbx.exe .
.\bin\blbx.exe check examples/language_tour.bx
.\bin\blbx.exe run examples/language_tour.bx
```

On macOS/Linux, build with `go build -o bin/blbx .`, then use `./bin/blbx`.
Go is required to build, but not to run the executable.

## Documentation

- [Complete documentation index](docs/README.md)
- [Language syntax and singleton methods](docs/language.md)
- [Standard library](docs/standard-library.md)
- [Async and await usage](docs/async.md)
- [CLI, diagnostics, and release builds](docs/cli.md)
- [Error codes](docs/error-codes.md)

## Status

The current default version is `0.1.0-dev`. BLBX is an alpha: variable scope,
user-function parameter validation, and indexed assignment have known limits.
Runtime errors cannot yet be caught. The native checker validates syntax and
conservatively detects undefined names before execution, including inside
uncalled branches and functions.

Runnable examples are in `examples/`. Automated tests were removed at the
project owner's request; build/vet and example checks do not replace a regression
suite. See the CLI guide for validation commands and release versioning.
