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

- [Complete documentation index](docs/README.md)
- [Language syntax and singleton methods](docs/language.md)
- [Exceptions, interfaces, operators, tuples, and inheritance](docs/features.md)
- [Standard library](docs/standard-library.md)
- [Files, OS, JSON, networking, and security](docs/system-library.md)
- [Async and await usage](docs/async.md)
- [CLI, diagnostics, and release builds](docs/cli.md)
- [Error codes](docs/error-codes.md)

## Status

The current default version is `0.1.0-dev`. BLBX is an alpha: variable scope,
user-function parameter validation, and numeric overflow handling have known limits.
Runtime failures can be caught with callback-style `try`. The native checker validates syntax and
conservatively detects undefined names before execution, including inside
uncalled branches and functions.

Runnable examples are in `examples/`. Automated tests were removed at the
project owner's request; build/vet and example checks do not replace a regression
suite. See the CLI guide for validation commands and release versioning.
