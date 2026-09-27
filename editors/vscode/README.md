# BLBX for Visual Studio Code

Syntax highlighting for `.bx` files, including comments, multiline strings and
escapes, `f"..."` strings with `{expression}` interpolation, numbers, booleans, null, imports, classes, interfaces, function names,
callback control flow, and BLBX operators. Also provides comment toggling,
bracket matching, and automatic bracket and quote closing.

Colors follow your active VS Code theme. This extension provides syntax
highlighting, not diagnostics, completion, or a language server.

## Install

From this directory, with Node.js and npm installed:

```sh
npm run package
code --install-extension blbx-language-0.1.0.vsix
```

Alternatively, run **Extensions: Install from VSIX...** in VS Code and select
the generated file. Open a `.bx` file; its language mode should be **BLBX**.
If the file is already open, use **Developer: Reload Window** after installing.

The manifest currently uses publisher ID `JoshuaWhitfield`, making the extension
identifier `JoshuaWhitfield.blbx-language`. Marketplace publication requires
access to that publisher, or changing `publisher` to an ID you own and rebuilding.
This repository's local package does not by itself confirm Marketplace publication.

## Develop

Open this directory as a VS Code workspace and press **F5** to launch the
extension development host with the language tour. No compilation is required.
Use **Developer: Inspect Editor Tokens and Scopes** to inspect highlighting.

The grammar follows the project's lexer and parser. Strings use double quotes
and can span lines; comments use `//` or `/* ... */`. Arithmetic and comparisons
mostly use method calls such as `value.add(1)` and `value.eq(2)`.

Formatted strings can span lines. Their `f` prefix, string text, and embedded
expressions receive distinct scopes; `{{` and `}}` are highlighted as escapes.
Interpolation also supports nested objects, blocks, and formatted strings.

VS Code's [syntax highlighting guide](https://code.visualstudio.com/api/language-extensions/syntax-highlight-guide)
explains the TextMate grammar format used here.
