# BLBX documentation

These references describe the current interpreter and distinguish supported behavior from implementation limits.

| Guide | Covers |
| --- | --- |
| [Language syntax](language.md) | Literals, variables, functions, control flow, objects, destructuring, classes, imports, and singleton methods |
| [Standard library](standard-library.md) | Every built-in module, function, result type, and operational limit |
| [Async and await usage](async.md) | Task creation, waiting, results, isolation, output, errors, and concurrency examples |
| [CLI and diagnostics](cli.md) | Building, running, checking, JSON diagnostics, exit codes, and release builds |
| [Error-code reference](error-codes.md) | All diagnostic codes and their meanings |

Start with the language guide, then run the examples from the repository root:

```powershell
.\bin\blbx.exe run examples/language_tour.bx
.\bin\blbx.exe run examples/async_tasks.bx
.\bin\blbx.exe run examples/standard_library.bx
```

BLBX uses `task.spawn()` and `task.await()` from `std.task`; there are no `async` or `await` keywords.
