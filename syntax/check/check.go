// Package check validates BLBX source without executing it or loading imports.
package check

import (
	"blbx_lang/syntax/diagnostic"
	"blbx_lang/syntax/ir"
	"blbx_lang/syntax/lexer"
	"blbx_lang/syntax/parser"
	"sort"
	"strings"
	"unicode/utf8"
)

func Source(file, source string) ([]parser.Node, []diagnostic.Diagnostic) {
	diagnostics := []diagnostic.Diagnostic{}
	if !utf8.ValidString(source) {
		return nil, append(diagnostics, diagnostic.Diagnostic{File: file, Line: 1, Column: 1, EndLine: 1, EndColumn: 1, Severity: "error", Code: diagnostic.InvalidUTF8, Message: "source is not valid UTF-8"})
	}
	var l lexer.Lexer
	l.Set(source)
	l.Tokenize()
	diagnostics = append(diagnostics, l.Diagnostics...)
	delimiters := checkDelimiters(l.Get())
	diagnostics = append(diagnostics, delimiters...)
	var p parser.Parser
	p.Set(l.Get())
	if len(delimiters) == 0 {
		p.Parse()
	}
	diagnostics = append(diagnostics, p.Diagnostics...)
	// Missing commas retain a usable tree; other syntax errors may discard
	// declarations, so avoid misleading name errors from those partial trees.
	usable := true
	for _, d := range diagnostics {
		if d.Code != diagnostic.MissingComma || !strings.HasPrefix(d.Message, "expected ',' between") {
			usable = false
		}
	}
	if usable {
		diagnostics = append(diagnostics, checkNames(ir.Lower(p.Get()), l.Get())...)
	}
	for index := range diagnostics {
		diagnostics[index].File = file
	}
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Line != diagnostics[j].Line {
			return diagnostics[i].Line < diagnostics[j].Line
		}
		return diagnostics[i].Column < diagnostics[j].Column
	})
	return p.Get(), diagnostics
}
