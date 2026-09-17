# BLBX import CLI

A small command language implemented entirely in BLBX: source → lexer tokens →
parsed command → execution. This is an import test project, not a BLBX compiler.

From the repository root:

```powershell
.\bin\blbx.exe run import_cli/main.bx
.\bin\blbx.exe run import_cli/demo.bx
```

Try these commands in the interactive CLI:

```text
echo "hello world"
add 12 30
tokens echo "two words"
parse echo "two words"
exit
```

`tokens` displays token records; `parse` displays the parsed command as JSON.
The lexer handles whitespace, double-quoted text, empty quoted strings, and
backslash escapes inside quotes. Escaping quotes/backslashes removes the escape
backslash; this command language does not interpret `\n` as a newline.
Unknown commands, bad numbers, and unterminated strings print messages and leave
the CLI running. An empty input line or EOF also exits.

```text
import_cli/
  main.bx                 interactive entry point
  demo.bx                 scripted smoke test
  cli/
    __init__.bx           package marker and prompt
    app.bx                command dispatch and input loop
  frontend/
    __init__.bx           package marker and shared lexer/parser bindings
    lexer.bx              characters → Token instances
    parser.bx             tokens → command object
  models/
    __init__.bx           package marker and Token binding
    token.bx              Token class
```

Each imported directory must contain `__init__.bx`. It may be empty. Importing
the directory itself loads that file; importing a dotted child requires its
parent directories to have the marker. Imports resolve from the entry script's
directory, so modules use names such as `frontend.lexer`, not
`import_cli.frontend.lexer`.

There are no export declarations or underscore visibility rules. All module
bindings can be accessed with `import` or `from ... import ...`. The package
initializers above import convenient names into their own namespace using
ordinary imports. Module execution is cached within an interpreter.
