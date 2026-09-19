# Async and await usage

[Documentation index](README.md) · [Language](language.md) · [Standard library](standard-library.md)

BLBX runs asynchronous work through `std.task`. There are no `async` or `await`
keywords. Ordinary functions become concurrent work when passed to `task.spawn`.
`task.await(handle)` waits for that work and returns its result.

## Quick start

```text
import std.task as task
import std.time as time

work = (number) => {
    time.sleep(100)
    return number.mul(2)
}
first = task.spawn(work, 21)
second = task.spawn(work, 10)
print(task.await(first))  // 42
print(task.await(second)) // 20
```

Both jobs start before either is awaited, so their waits overlap. Calling
`task.await(task.spawn(work, 21))` and then doing the same for the second job
would wait for each job before starting the next.

## API

| Function | Arguments | Result |
| --- | --- | --- |
| `task.spawn(function, ...arguments)` | One function, then its positional arguments | New `task` handle |
| `task.await(handle)` | Exactly one task handle | Copy of the function's result, or an error |
| `task.done(handle)` | Exactly one task handle | Boolean indicating completion, including failure |

`spawn` accepts BLBX and native function values; a class definition is not a
function argument. To construct a class in a worker, wrap construction in a
function. `typeof(handle)` returns `"task"`, and printing a handle shows `<task>`.
Handles do not expose `.await()` or `.done()` methods themselves: call the
functions on the imported `task` module. Handles cannot be JSON-serialized.

Argument expressions are evaluated in the caller before spawning. Pass the
function itself, not the result of calling it:

```text
import std.task as task
square = (value) => { return value.mul(value) }
job = task.spawn(square, 9)
print(task.await(job)) // 81
```

`task.spawn(square(9))` calculates 81 immediately and then fails because 81 is
not a function. Extra/missing arguments to the worker obey that worker's own
arity rules: BLBX functions are permissive, native functions are strict.

## Captured state and isolation

At spawn time, the runtime snapshots the function, captured scopes, and supplied
arguments before starting the worker. Mutable BLBX objects and arrays are not
shared with the caller. The snapshot preserves internal aliases and cycles,
including recursive functions, closures, and class instances. Existing task
handles are shared completion handles rather than copied workers. Network
resources are also shared external handles; closing one affects every holder.

```text
import std.task as task
state = {"count": 1}
change = () => {
    state.count = 99
    return state
}
job = task.spawn(change)
result = task.await(job)
print(state.count)  // 1
print(result.count) // 99
```

Changes made by the caller after spawning do not update captured worker values.
Worker writes do not update caller globals. Return results explicitly instead
of using captured objects as shared counters. Each worker has its own interpreter
and module cache; source imports performed inside workers can initialize again.
Already captured module values are copied with the captured environment.

External resources remain shared: two workers writing the same file can interfere,
and network requests or child processes have real effects. State isolation does
not coordinate those operations. Large captured graphs take time and memory to
copy; `spawn` returns after snapshotting, not before it.

## Awaiting and repeat awaits

Awaiting blocks the current caller until completion. Other spawned tasks can
continue running. Results preserve their BLBX type: primitives, arrays, objects,
class instances, closures, and task handles can be returned.

Each await returns a fresh snapshot of the stored result:

```text
import std.task as task
make = () => { return {"count": 1} }
job = task.spawn(make)
first = task.await(job)
first.count = 20
second = task.await(job)
print(second.count) // 1
```

The task does not rerun when awaited again. Its printed output is also not replayed.
Returning an inner task handle does not automatically await that inner task;
the caller must await it explicitly if its result is needed.

## Completion checks

`done` never waits and does not return a task's result or raise its stored failure.
A true result means the task finished, not necessarily that it succeeded. Call
`await` to retrieve the result or observe an error.

```text
import std.task as task
import std.time as time
work = () => { time.sleep(20) return "ready" }
job = task.spawn(work)
while(task.done(job).not(), () => {
    time.sleep(5)
})
print(task.await(job))
```

Polling is optional. A direct await is simpler when there is no other work to do.

## Output and input

Worker `print` and `std.os.write_stdout` output is buffered.
`std.os.write_stderr` has a separate buffer replayed to stderr at await.
Script arguments are copied into workers; OS environment changes are process-wide. The first await emits that buffer once,
before returning the result or propagating an error. Await order therefore
controls when buffered output is displayed; completion order alone does not.

```text
import std.task as task
say = (message) => { print(message) return null }
first = task.spawn(say, "first")
second = task.spawn(say, "second")
task.await(second) // prints second
task.await(first)  // prints first
task.await(first)  // no additional output
```

If multiple callers await the same handle, whichever awaits first consumes its
output. In nested tasks, that output enters the awaiting worker's own buffer,
then appears when that worker is awaited. There is no live output streaming or
buffer size limit for task output yet; avoid unbounded printing in workers.
Workers have empty standard input. `input()` there reaches EOF and returns an
empty string; it does not prompt interactively for terminal input.

## Errors

Static checks run before execution. An unresolved variable in a worker function
can stop the entire file before spawning, even if that function would never run.
Dynamic errors that occur while a worker runs are stored and surfaced by await.

This example intentionally fails:

```text
import std.task as task
work = () => { return [1].not() }
job = task.spawn(work)
task.await(job)
```

The error is:

```text
runtime BX4002: std.task.await: method "not" is not available on array
```

Await preserves the original category/code and adds context. Missing arguments
use BX4003, invalid argument types use BX4004, and an unexpected recovered worker
failure uses BX4013. See [all error codes](error-codes.md).
Awaited failures can be caught with `try`:

```text
import std.task as task
job = task.spawn(() => { throw("worker failed") })
print(try(() => { return task.await(job) }, (error) => { return error.value }))
```

The handler gets code, message, phase, and a snapshot of the user-thrown value.
Uncaught awaited errors stop execution; unawaited failures are not reported.
Repeated awaits can catch the same stored failure again.

## Nested tasks

Workers can spawn and await more workers:

```text
import std.task as task
outer = (number) => {
    inner = task.spawn((value) => { return value.mul(2) }, number)
    return task.await(inner)
}
print(task.await(task.spawn(outer, 6))) // 12
```

Each spawn uses a new snapshot. A parent worker should await the children whose
results or buffered output it needs.

## Multiple jobs and bounded batches

Store handles in an array, then await them in a second loop:

```text
import std.task as task
jobs = []
for([1, 2, 3], (cursor) => {
    jobs = jobs.append(task.spawn((n) => { return n.mul(n) }, cursor.elem()))
})
results = []
for(jobs, (cursor) => {
    results = results.append(task.await(cursor.elem()))
})
print(results) // [1, 4, 9]
```

There is no automatic pool or concurrency limit. For large input, process fixed
batches rather than spawning a task for every item:

```text
import std.task as task
items = [1, 2, 3, 4, 5]
results = []
offset = 0
while(offset.lt(items.length()), () => {
    jobs = []
    for(items.slice(offset, offset.add(2)), (cursor) => {
        jobs = jobs.append(task.spawn((n) => { return n.mul(n) }, cursor.elem()))
    })
    for(jobs, (cursor) => {
        results = results.append(task.await(cursor.elem()))
    })
    offset = offset.add(2)
})
print(results) // [1, 4, 9, 16, 25]
```

This limits the example to two active workers per batch. A batch waits for all
its results before the next starts; there is no built-in `await_all`, `race`, or
completion-order iterator.

## Native I/O in workers

Native calls block their worker while independent tasks can continue. For
example, two HTTP requests can overlap by spawning a wrapper around
`std.networking.get`. Return the response objects and inspect `status` and `body`
after awaiting. HTTP non-2xx status codes are responses; transport failures are
errors. HTTP and process calls retain their individual 15-second limits.

File functions captured from `std.files` keep the importing file's base directory.
Use separate output paths for independent writers, or await each write before
starting a dependent read.

## Lifetime and current limits

- Await every job needed before the main script exits. Tasks do not keep the CLI alive.
- There is no task cancellation, deadline, timeout argument, or scheduler API.
- There is no shared-memory channel, mutex, atomic operation, or worker pool API.
- A long-running or nonterminating task can make await wait indefinitely.
- Snapshot copying, output buffering, and the number of workers are not bounded automatically.
- Execution uses Go goroutines; ordering and performance depend on the workload.

Use ordinary calls when concurrency adds no useful overlap or when operations
must happen in a strict sequence.
