# BLBX language reference

[Documentation index](README.md) · [Standard library](standard-library.md) · [Asynchronous tasks](async.md)

This reference describes the current `0.1.0-dev` interpreter. Examples use BLBX,
regardless of the syntax coloring chosen by an editor. Save runnable examples as
UTF-8 `.bx` files and execute `blbx run file.bx`.

## Contents

- [Source and tokens](#source-and-tokens)
- [Values and variables](#values-and-variables)
- [Expressions and operators](#expressions-and-operators)
- [Functions and closures](#functions-and-closures)
- [Control flow](#control-flow)
- [Objects and indexing](#objects-and-indexing)
- [Imports and packages](#imports-and-packages)
- [Destructuring assignments](#destructuring-assignments)
- [Classes and self](#classes-and-self)
- [Type inspection and singletons](#type-inspection)
- [String and array methods](#string-and-array-singleton-methods)
- [Errors](#undefined-references)
- [Current limitations](#current-limitations)

## Source and tokens

Whitespace and indentation are not structural. Use braces for bodies and one
statement per line for clarity. Semicolons are not statement terminators.
Commas are required between arguments, parameters, array elements, and object
fields, even across lines. Trailing commas are accepted in those lists;
leading commas and empty entries are errors.

```text
// Single-line comment
/* A block comment
   can span lines. */
message = "Hello, BLBX"
print(message)
```

Identifiers are case-sensitive and use letters, digits, and underscores; do not
start a name with a digit. Reserved syntax words are `class`, `extends`, `self`,
`return`, `assert`, `import`, `from`, and `as`. `true`, `false`, and `null` have
special literal behavior. Avoid assigning to literal names or built-in names.
`func` is an ordinary identifier; it is not required to declare a function.
`if`, `for`, and `while` use call-shaped control-flow syntax.

### Literals

| Kind | Example | Notes |
| --- | --- | --- |
| Null | `null` | Explicit absence of a value |
| Boolean | `true`, `false` | Lowercase |
| Integer | `42` | Signed 64-bit runtime representation |
| Float | `3.25` | 64-bit floating point |
| String | `"hello"` | Double quotes only |
| Array | `[1, "two", null]` | Ordered, heterogeneous elements |
| Object | `{"name": "Ada", "age": 36}` | String-keyed fields |

Numeric source literals use decimal digits and an optional fractional part.
Unary minus, exponent notation, hexadecimal, and numeric separators are not
supported in source literals. Use `(0).sub(3)` or `("-3").to_int()` for a
negative integer; string-to-float conversion accepts more formats than literals.
Avoid out-of-range numeric literals: literal overflow currently falls back to
zero instead of producing a range diagnostic. Integer arithmetic does not check
overflow.

Strings support `\n`, `\r`, `\t`, `\b`, `\f`, `\"`, and `\\`. For an unknown
escape, the backslash is dropped and the following character is retained.
There is no interpolation or `\uXXXX` escape decoding; write Unicode directly.
Actual line breaks may occur within a quoted string. String methods count
Unicode code points rather than display graphemes.

## Values and variables

```text
count = 1
count = count.add(1)
items = [count, true, null]
print(items)
```

Assignment creates or updates a binding. There are no `let`, `const`, static type
annotations, or immutable-binding declarations. Values carry their runtime type,
so a variable can later hold a different type. Arrays, objects, functions,
classes, and tasks are values. A class instance reports type `object`.

### Scope and sharing

Functions capture their surrounding environment. Parameters belong to the call
scope. Assignment updates the nearest existing binding; if no binding exists in
that scope chain, it creates one in the root scope. Consequently a newly assigned
name inside a function can become a module/global binding. A brace block does
not provide conventional block-local declarations. Avoid reusing scratch names
across recursive calls; parameters are the reliable per-call bindings.

Objects normally share their fields when passed or assigned. Updating an object's
field through an alias affects the original object. Array transformation methods
return new shallow arrays; nested objects remain shared. Task boundaries use
separate snapshots, as described in [async usage](async.md).

### Truthiness

| Value | False when |
| --- | --- |
| `null` | Always |
| Boolean | `false` |
| Integer or float | Zero |
| String | Empty |
| Array | Empty |
| Object, function, class, task | Never, including empty objects |

Conditions use truthiness. `.to_bool()` converts it to a boolean explicitly.
Logical singleton methods require boolean receivers and evaluated arguments.

## Expressions and operators

Calls, dot access, indexing, and method chains compose left to right.
Parentheses group values and enable singleton-style spelling. Parenthesized comma lists create tuples: `(1, 2)`, `(1,)`, and `()`.
`(value)` remains that value. Tuples have a distinct `tuple` type and reject indexed writes.

| Operation | Syntax | Result |
| --- | --- | --- |
| Add/subtract/multiply | `a.add(b)`, `a.sub(b)`, `a.mul(b)` | Integer for two integers; otherwise float |
| Divide | `a.div(b)` | Float; zero divisor returns `null` |
| Remainder | `a.mod(b)` | Integer operands; zero divisor returns `null` |
| Compare | `a.lt(b)`, `.lte(b)`, `.gt(b)`, `.gte(b)` | Boolean; numbers or two strings |
| Equality | `a.eq(b)`, `a.neq(b)` | Scalar equality; integers/floats compare numerically |
| Boolean operations | `a.and(b)`, `a.or(b)`, `a.not()` | Boolean; `and`/`or` short-circuit |
| Infix multiply | `a * b` | Supported legacy arithmetic syntax |

Only `*` is currently an infix arithmetic operator. Chained calls bind more
tightly than `*`, which associates left to right. Use methods for other
arithmetic; `+`, `-`, `/`, `%`, `==`, `&&`, and `||` are not source operators.
Use numeric operands with `*`, or provide a left-hand multiplication overload. `=` assigns and `=>` introduces a function.

Arrays, ordinary objects, functions, classes, and task handles have no built-in
deep or identity equality; `.eq()` returns false unless an object overloads it.
Tuples compare elements recursively using those equality rules. Mixed-type
ordering returns false. Invalid numeric method operands return `null`.

```text
print((6).mul(7))                 // 42
print((5).div(2))                 // 2.5
print((2).lt(3).and((4).gt(1)))    // true
print([].to_bool().not())         // true
```

## Functions and closures

```text
greet = (name) => {
    return "Hello, ".concat(name)
}
print(greet("Ada"))

optional = (value = null) => {
    return value
}
print(optional()) // null
```

Use `(parameters) => { statements }`; zero parameters use `()`. Always use a
brace body: although expression bodies can parse, the runtime currently executes
block bodies. Calls use positional arguments. There are no keyword arguments,
rest parameters, or parameter destructuring.

User-function arity is permissive: missing arguments use defaults (or `null`),
and extra arguments are evaluated then ignored. Native standard-library calls
and singleton methods validate their arity. Defaults are evaluated when the
function is created, so mutable default values can be shared across calls.
A single defaulted parameter, as above, is supported; defaults within a
multi-parameter list are not reliably bound in the current runtime. Use explicit
arguments for multi-parameter functions.

`return value` exits the current function. Use `return null` for an explicit
empty result; a bare `return` is not supported by the parser. Without `return`,
a function evaluates to its body's last statement value, or `null` for an empty
body. Returning inside a callback exits that callback, not the enclosing function.

Functions are first-class: pass them as arguments, store them in collections,
and return them. Captured parameter bindings remain available after a call ends.

```text
make_adder = (base) => {
    return (value) => { return base.add(value) }
}
add_ten = make_adder(10)
print(add_ten(5)) // 15
```

Functions can refer to themselves or to later declarations. Static checking
permits those references, but the referenced value must exist when called.

## Control flow

### `if(condition, then_branch, else_branch)`

The else branch is optional. Only the selected branch is evaluated. Passing a
function invokes it with no arguments; a selected ordinary expression supplies
its value directly. The `if` expression returns the selected branch's result,
or `null` if there is no matching branch.

```text
value = 10
label = if(
    value.gt(5),
    () => { return "large" },
    () => { return "small" }
)
print(label) // large
```

### `for(array, callback)`

The callback receives a cursor, not the array element itself.

| Cursor method | Result |
| --- | --- |
| `elem()` | Current element |
| `idx()`, `index()` | Zero-based index |
| `step()` | One-based iteration count |
| `continue()` | Skip the remainder of this iteration |
| `break()` | Exit the loop |

```text
for([10, 20, 30], (cursor) => {
    if(cursor.elem().eq(20), () => { cursor.continue() })
    print(cursor.elem())
})
```

`for` accepts arrays, tuples, strings, and custom iterators. It returns the last
callback result or null for empty iteration. Invalid iterable/callback values
raise errors. See [iterators](features.md#iterator-protocol). Cursor control methods apply only to actual loop cursors.

### `while(condition_expression, body)`

The condition expression is reevaluated each iteration. Supply a boolean-producing
expression or an explicit predicate call, not a bare predicate function: a
function value itself is truthy. The body may be a zero-argument callback.

```text
counter = 0
while(counter.lt(3), () => {
    print(counter)
    counter = counter.add(1)
})
```

`while` returns the last body result, or `null` if it never runs. There is no
standalone `break`/`continue` keyword or while-specific cursor. Use the condition
to terminate it. Control-call arity is not yet consistently validated; use the
signatures shown here.

### Recursion over nested arrays

```text
recurse = (items) => {
    for(items, (cursor) => {
        if(cursor.elem().typeof().eq("array"),
            () => { recurse(cursor.elem()) },
            () => { print(cursor.elem()) }
        )
    })
}
recurse([1, [2, 3], [4, [5]]])
```

### `assert(value)`

This is currently a truthy early-exit construct, not a conventional testing
assertion. A truthy value stops the current evaluation sequence with that value;
a falsey value returns `null` and continues. Function and branch boundaries
consume this control signal. It does not throw an assertion error. Prefer
explicit `if` and `return` for new code unless you need this existing behavior.

### Console built-ins

`print(values...)` writes each value on its own line and returns `null`.
`input(prompt)` optionally prints the first argument as a prompt, reads one line,
and strips its line ending. `input()` has no prompt; EOF can return an empty
string. `typeof(value)` returns the runtime type name and requires one argument.

## Objects and indexing

```text
person = {"name": "Ada", "age": 36}
person.age = person.age.add(1)
print(person.name)
print(person["age"])
items = ["first", "second"]
print(items[0])
```

Literal field names must be quoted strings followed by `:`. Duplicate literal
keys overwrite earlier values; duplicate keys in destructuring patterns are
errors. Object field values evaluate in order, with `self` available.
`{}` on an assignment's right side creates an empty object. A standalone empty
brace block evaluates to `null`; braces are also used for statement bodies.
A body containing only quoted-key fields evaluates as an object literal.

Use dot or bracket assignment to add or change fields on existing objects.
Integer array assignment updates an existing slot; out-of-range writes raise
BX4014. Strings and tuples reject writes. Bracket reads support integer array,
tuple, and string indices, and string object keys. Out-of-range
array reads, including negative indices, return `null`; missing object fields
are errors. String bracket indices count Unicode code points and accept negative indices
from the end, just like `.char_at(index)`.

### self in plain objects

Inside an object literal, `self` refers to the object being created. Fields
evaluate in order, so an initializer can read fields already initialized.
Inside a method, `self` refers to the object receiving the call.

```text
counter = {
    "value": 0,
    "increment": () => {
        self.value = self.value.add(1)
        return self.value
    }
}
print(counter.increment()) // 1
saved = counter.increment
print(saved()) // 2; the extracted method retains its receiver
```

Dot and bracket method access both bind `self`. A nested object literal has its
own `self`; capture the outer receiver in another variable if needed. `self`
cannot be reassigned or used as a parameter, and using it without a receiver
is an error. Class receivers are covered under [Classes and self](#classes-and-self).

## Imports and packages

```text
import std.math as math
from std.strings import upper, lower as down
print(math.sqrt(81))
print(upper("blbx"))
```

Import paths are dotted names, not quoted file paths. `import pkg.module`
binds a namespace rooted at `pkg`; `import pkg.module as alias` binds the module
directly. `from pkg.module import name as alias` binds selected values.
There are no exports, wildcard imports, or relative-import-dot syntax.
Every module binding is accessible, including underscore-prefixed names.

A source package can use this layout:

```text
project/
  main.bx
  helpers/
    __init__.bx
    greeting.bx
```

`helpers/greeting.bx`:

```text
hello = (name) => { return "Hello, ".concat(name) }
```

`main.bx`:

```text
import helpers.greeting as greeting
print(greeting.hello("Ada"))
```

`helpers/__init__.bx` can be empty. Every directory crossed by a dotted import
must have that marker. `import helpers` runs `helpers/__init__.bx`; importing a
child checks the ancestor marker but does not automatically execute it.
Paths resolve from the entry script's directory, including imports issued by
nested modules. Resolution tries `path.bx` before `path/__init__.bx`.

Modules run in their own root scope and are cached per interpreter. Repeated
imports share module object state. Cyclic imports may expose an incompletely
initialized module; do not depend on values before their initialization.
There is no package manager, dependency resolver, configurable search path,
or automatic installation. Built-in standard modules take precedence over
matching source paths. File I/O uses separate [path rules](standard-library.md#files--stdfiles).

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

Inheritance uses `class Child extends Parent { ... }` or multiple comma-separated
bases, such as `class Child extends First, Second { ... }`. Each parent must
already be a class; imported parents can use a dotted name such as `models.Base`.
Fields initialize once per ancestor in reverse C3 method resolution order,
with higher-priority declarations overriding lower-priority ones. Mutable defaults remain separate per instance. Inherited methods keep
their defining module's lexical scope, while `self` refers to the child instance.

A child without a constructor inherits the nearest ancestor's constructor. A
child with its own constructor replaces it; parent constructors are not called
automatically. Use `super(arguments...)` for the next constructor and
`super.method(arguments...)` for the next implementation in the instance MRO.

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

### self in classes

For a class instance, `self` refers to that instance: use `self.field` to read
or write its attributes and `self.method()` to call its methods.
`self` refers to the receiver and is available inside constructors, methods,
nested callbacks, and object-literal initializers. It cannot be reassigned or
used as a parameter; accessing it without a receiver is an error.
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
overflow, and nonfinite string-to-float results return `null`. Unsupported
receiver types and invalid argument counts raise runtime errors. String conversions do not trim whitespace.
Numeric arithmetic methods and string `concat()` retain their type-specific
behavior. Existing calls on variables and unparenthesized values remain supported.

Use `typeof(value)`, the singleton property `value.typeof`, or the method
`value.typeof()` to obtain a type
name as a string. Both support `null`, `boolean`, `integer`, `float`, `string`,
`array`, `tuple`, `object`, `function`, `class`, `interface`, `task`, and `resource`.

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
side_effect = () => { print("called") return true }
print((false).and(side_effect()))        // false; RHS is not evaluated
```

On booleans, `and` and `or` require one argument; `not` requires none. `and`
skips its argument when the receiver is false; `or` skips it when true.
An evaluated argument must be boolean. Wrong arity or a non-boolean evaluated
argument raises a runtime error. Method chains execute left to right. Ordinary
object methods named `and`, `or`, or `not` retain normal eager call behavior.
Non-boolean receivers without custom methods raise a runtime error. Use
`.to_bool()` to convert explicitly: `[].to_bool()` is false and
`[1].to_bool()` is true. `.not()`, `.and()`, and `.or()` remain boolean-only.

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
index. Invalid argument types on sequence methods return `null`; invalid
argument counts raise runtime errors.

## Undefined references

Using an undefined variable, calling an undefined function/method, reading a
missing object member (dot or string-key indexing), or calling a non-callable
value raises a runtime error. Member assignment requires an existing object:
use `obj = {}` before `obj.field = value`. New variables and new fields on
existing objects remain valid.

Explicit `null` values are valid and distinct from undefined names. Failed
conversions and out-of-range array reads still return `null`. Existing loop cursor
accessors such as `.elem()` remain supported. Errors currently stop execution
and make `blbx run` exit with code 1 unless caught using `try(work, handler)`.
Use `throw(value)` for a user-defined failure. See [exceptions](features.md#exceptions).
Both `check` and `run` check statically unresolved names, including names in
uncalled functions. Dynamic member access and actual binding availability are
checked at runtime. See [diagnostics](cli.md#current-checks-and-scope).


See [additional language features](features.md) for interfaces, operator hooks,
tuples, iterators, exceptions, and multiple-inheritance examples.
## Current limitations

- Immutable bindings and static types remain intentionally omitted.
- Interfaces validate method names and arity, not parameter/return types.
- Exceptions use callbacks; there is no block-style try/catch or finally.
- Function scoping, parameter validation, and numeric overflow handling remain incomplete.
- No async/await keywords: use [std.task](async.md).
- No formatter, debugger, package manager, or general REPL in the CLI.
- Go manages runtime memory; BLBX has no manual allocation/free API.

These are implementation limits, not guarantees about future syntax.
