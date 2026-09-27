# Additional language features

[Language reference](language.md) · [Async usage](async.md) · [Error codes](error-codes.md)

Immutable bindings and static types remain intentionally omitted. Existing
variables stay dynamically typed. The following features are implemented.
Run `blbx run examples/language_features.bx` for a combined demonstration.

## Exceptions

`try(work, handler)` calls a zero-argument work function. On success it returns
that function's result. On failure it calls the handler with an error object
and returns the handler's result. Both arguments must be functions.

```text
result = try(
    () => { throw("invalid input") },
    (error) => { return error.value }
)
print(result) // invalid input
```

`throw(value)` accepts any BLBX value and raises BX4015. The handler receives:

| Field | Meaning |
| --- | --- |
| `code` | Original diagnostic code |
| `message` | Human-readable failure message, including wrapper context |
| `phase` | `runtime` or `syntax` |
| `value` | Snapshot of a user-thrown value; null for ordinary runtime errors |

Native-library errors, invalid indexing, iterator failures, and awaited task
errors are catchable. Failures in the handler propagate to an enclosing `try`.
To throw another value from a handler, call `throw` again. Mutations made before
an error are not rolled back. A return inside either callback returns from that
callback; it does not return from the caller of `try`.

```text
print(try(
    () => { return [1].not() },
    (error) => { return error.code }
)) // BX4002
```

Entry-file syntax and static undefined-name checks happen before execution and
cannot be caught by code in that file. Errors arising when an imported source
module is loaded inside work can be caught. `try` only catches failures while
calling work, not while evaluating the expressions that supply its arguments.

There is no block-style `try { } catch { }`, `finally`, or separate `catch`
keyword. The second callback is the catch handler. Process termination and fatal
Go runtime failures are not recoverable BLBX exceptions.

## Indexed assignment and string indexing

```text
items = [10, 20]
items[1] = 99
object = {}
object["name"] = "Ada"
print(items)        // [10, 99]
print(object.name)  // Ada
print("hello"[1])   // e
```

Array indices must be nonnegative integers within the existing array; invalid
writes raise BX4014 and do not grow it. Object indices must be strings and can
create new fields. Nested targets work, for example `items[0]["name"] = "Ada"`
when the first item is an object. The assigned value and each index expression
are evaluated once, with the right-hand value evaluated before the target.
Array aliases observe indexed mutations.

`items.extend(other_array)` also mutates the shared array, adding elements and
returning `null`. Aliases observe its new length as well as its elements.
`append` and `concat` continue to return new shallow arrays. See
[array_extend.bx](../examples/array_extend.bx).

String reads count Unicode code points. Negative string indices count from the
end; out-of-range reads return null. Strings cannot be modified by indexing.
Array/tuple out-of-range reads return null; their negative indices are not
normalized. Missing object-key reads raise BX3002.

## Tuples

```text
empty = ()
one = (1,)
pair = (1, 2)
print(pair.typeof()) // tuple
print(pair[0])       // 1
print(pair.eq((1, 2))) // true
```

A comma distinguishes a one-element tuple from grouping: `(1)` is an integer.
`()` is now an empty tuple rather than an array. Tuples are shallowly immutable:
their slots cannot be reassigned, but an object contained in a tuple remains
mutable. This does not introduce immutable variable bindings.

Tuples support `length`/`length()`, `is_empty()`, `first()`, `last()`, `join()`,
`slice()`, `reverse()`, `contains()`, and `index_of()`, plus universal methods
such as `typeof`, `to_str`, and `to_bool`. Slicing and reversing return tuples;
there is no tuple `append` or `concat` method. Empty tuples are falsey.
Tuples compare element-by-element using existing equality rules. Array-pattern
destructuring and `for` accept tuples. JSON encodes them as arrays, so decoding
does not restore tuple identity. APIs explicitly requiring arrays still require
arrays unless documented otherwise.

## Iterator protocol

`for(iterable, callback)` now accepts arrays, tuples, strings, and custom objects.
String iteration yields one code-point string at a time. The callback always
receives the existing cursor with `elem()`, `idx()`, `step()`, `break()`, and
`continue()` methods.

A custom iterable defines `iter()` returning an iterator object. An iterator
must define `next()` returning one of:

- `{"done": false, "value": item}` to yield an item, including null.
- `{"done": true}` to finish.

An object with `next()` can also be passed directly to `for`. If both `iter`
and `next` exist, `iter()` is used. Class instances can implement this protocol.
The `done` field must be a boolean, and `value` is required when done is false.
Malformed results raise BX4016.

```text
class Count {
    (limit) => { self.limit = limit self.position = 0 }
    iter = () => { return self }
    next = () => {
        if(self.position.gte(self.limit),
            () => { return {"done": true} },
            () => {
                item = self.position
                self.position = self.position.add(1)
                return {"done": false, "value": item}
            }
        )
    }
}
for(Count(3), (cursor) => { print(cursor.elem()) }) // 0, 1, 2 on separate lines
```

Iteration is lazy: one next call per requested element; break does not request
another item. Reusing a consumed iterator does not reset it. Implement `iter()`
to return a fresh iterator if the collection needs repeatable traversal.
There is no implicit iterator cleanup/close callback or generator `yield` syntax.
Inside a newly returned object literal, `self` refers to that literal; copy an
outer receiver field into a variable first, as in the example.

Array iteration reads the current array length on each step, so elements added
with `extend` during iteration can also be visited. `while` provides the same
cursor controls and counters, with `elem()` returning `null`; see
[while_cursor.bx](../examples/while_cursor.bx).

## Interfaces

Interfaces declare required method names and positional arities:

```text
interface Labelled {
    label()
}
class Token implements Labelled {
    (name) => { self.name = name }
    label = () => { return self.name }
}
print(Token("STRING").label())
```

A class can implement multiple comma-separated interfaces. Put `implements`
after `extends` when both are used. Inherited methods can satisfy contracts,
and descendants retain their ancestors' interface requirements. Construction
validates that each required method is a function with the declared parameter
count; a failure raises BX3012. Constructor side effects are not rolled back
if validation fails. Use BLBX-defined methods for declared parameter counts;
native functions do not expose their signatures through this interface check.

`typeof(Labelled)` is `interface`. Interfaces are not constructible, cannot be
JSON-serialized, and contain no fields or implementations. They enforce names
and arities, not parameter or return types. There are no static type annotations,
interface inheritance, overload signatures, or automatic structural declarations.

## Operator overloading

Object/class methods can provide operator hooks. Ordinary methods with the
operation name take precedence over its hook.

| Operation | Hook |
| --- | --- |
| `a.add(b)` | `__add__(b)` |
| `a.sub(b)` | `__sub__(b)` |
| `a.mul(b)` and `a * b` | `__mul__(b)` |
| `a.div(b)` | `__div__(b)` |
| `a.mod(b)` | `__mod__(b)` |
| `a.eq(b)`, `a.neq(b)` | `__eq__(b)`, `__ne__(b)` |
| `a.lt(b)`, `a.lte(b)` | `__lt__(b)`, `__le__(b)` |
| `a.gt(b)`, `a.gte(b)` | `__gt__(b)`, `__ge__(b)` |
| `a.and(b)`, `a.or(b)`, `a.not()` | `__and__(b)`, `__or__(b)`, `__not__()` |

```text
class Scale {
    (factor) => { self.factor = factor }
    __mul__ = (number) => { return self.factor.mul(number) }
}
print(Scale(3) * 4) // 12
```

Hooks use `self` and inherit like ordinary methods. They return ordinary BLBX
values; operand and result type conventions are the hook author's responsibility.
The left operand handles the operation; no reflected/right-hand hooks are used.
`__ne__` is not synthesized from `__eq__`. Built-in collection searches still
use built-in value equality and do not invoke object hooks.

Object logical hooks evaluate arguments eagerly, like ordinary method calls.
Boolean `.and()`/`.or()` retain their short-circuit behavior. Hooks do not add
new infix tokens: `*` remains the only infix arithmetic operator.

## Multiple inheritance and super

```text
class A { label = () => { return "A" } }
class B extends A { label = () => { return "B".concat(super.label()) } }
class C extends A { label = () => { return "C".concat(super.label()) } }
class D extends B, C { label = () => { return "D".concat(super.label()) } }
print(D().label()) // DBCA
```

BLBX computes C3 MRO, preserving local base priority and consistent ancestor
ordering. Duplicate bases and inconsistent orders raise BX4017. Field and method
initializers run once per class in reverse MRO. More-specific definitions override
less-specific ones. Each instance gets its own initialized mutable defaults.

`super.method()` starts lookup after the class that defined the currently bound
method, using the actual instance's MRO. It does not mean simply the first base.
Extracting `super.method` preserves the original receiver. A class method rebound
to a plain object cannot use super.

`super(arguments...)` invokes the next available constructor in that same order.
Parent constructors are not called automatically when a child declares one:

```text
class Base {
    (name) => { self.name = name }
}
class Child extends Base {
    (name) => { super(name) self.ready = true }
}
print(Child("Ada").name)
```

An absent child constructor inherits the first constructor found in its MRO.
When no further constructor exists, `super()` with no arguments is a no-op;
passing arguments is an error. Cooperative constructors must call super themselves
when they intend the chain to continue. Constructor return values never replace
the instance. Nested callbacks retain lexical access to super.

`super.name` can also read the declared value stored for an ancestor's field;
use `self.name` to read current instance state after later mutations. Super is
available only in class-method context, not as a general global function.
