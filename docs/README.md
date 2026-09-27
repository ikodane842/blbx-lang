# BLBX documentation

These references describe the current interpreter and distinguish supported behavior from implementation limits.

| Guide | Covers |
| --- | --- |
| [Language syntax](language.md) | Literals, variables, functions, control flow, objects, destructuring, classes, imports, and singleton methods |
| [Additional language features](features.md) | Exceptions, interfaces, overloads, tuples, indexing, iterators, multiple inheritance, and super |
| [Standard library](standard-library.md) | Every built-in module, function, result type, and operational limit |
| [Files, OS, JSON, networking, security](system-library.md) | Complete new APIs, handles, timeouts, HTTP servers, and cryptographic primitives |
| [Async and await usage](async.md) | Task creation, waiting, results, isolation, output, errors, and concurrency examples |
| [CLI and diagnostics](cli.md) | Building, running, checking, JSON diagnostics, exit codes, and release builds |
| [Error-code reference](error-codes.md) | All diagnostic codes and their meanings |
| [VS Code extension](../editors/vscode/README.md) | Syntax highlighting, formatted strings, installation, and development |
| [Black Box graph runtime](graph-runtime-design.md) | Graph construction and traversal, effects, classes, objects, and backend comparison |

Start with the language guide, then run the examples from the repository root:

```powershell
.\bin\blbx.exe run examples/language_tour.bx
.\bin\blbx.exe run examples/async_tasks.bx
.\bin\blbx.exe run examples/standard_library.bx
```

BLBX uses `task.spawn()` and `task.await()` from `std.task`; there are no `async` or `await` keywords.

## Focused examples

Run these from the repository root with `blbx run examples/<filename>`.
Several examples throw an error if their checks fail.

| Example | Demonstrates |
| --- | --- |
| [formatted_strings.bx](../examples/formatted_strings.bx) | Multiline interpolation, escaped braces, nested expressions, and evaluation order |
| [bitwise.bx](../examples/bitwise.bx) | Integer bit operations, signed and unsigned right shifts, and flags |
| [ord.bx](../examples/ord.bx) | Unicode code points and argument validation |
| [object_membership.bx](../examples/object_membership.bx) | String-key membership, including keys whose value is null |
| [array_extend.bx](../examples/array_extend.bx) | In-place extension, shared aliases, self-extension, and task isolation |
| [while_cursor.bx](../examples/while_cursor.bx) | Loop counters, break, continue, and nested control flow |
| [control_flow_returns.bx](../examples/control_flow_returns.bx) | Returns through control callbacks and statement-local assert results |
| [graph_execution.bx](../examples/graph_execution.bx) | Discarded-return effects, selected branches, recursion, object identity, and closures on both backends |
| [declarative_values.bx](../examples/declarative_values.bx) | Shared pure values, binding versions, effect ordering, mutation boundaries, and uncached user calls |

These additions are covered in the [language reference](language.md). `assert`
is a control-flow operation, not a test assertion: it exits the nearest executing
`if`, `for`, or `while`, even when passed a falsey value. An ordinary `return`
inside those callbacks exits the enclosing ordinary function, or stops the script
at global scope.
