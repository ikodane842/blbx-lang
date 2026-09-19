package check

import (
	"blbx_lang/syntax/diagnostic"
	"blbx_lang/syntax/ir"
	"blbx_lang/syntax/lexer"
	"fmt"
	"strings"
)

// This is a conservative name-presence check, not definite-assignment analysis.
// Scope.Set can create globals from inside functions, so all possible assignment
// bindings are collected before checking reads. Parameters remain lexical.
func checkNames(root ir.Node, tokens []lexer.LexerToken) []diagnostic.Diagnostic {
	globals := map[string]bool{"null": true}
	classes := map[string]ir.Node{}
	builtins := map[string]bool{"print": true, "input": true, "typeof": true, "if": true, "for": true, "while": true, "assert": true}
	builtins["try"], builtins["throw"] = true, true
	var bind func(ir.Node, map[string]bool)
	bind = func(n ir.Node, names map[string]bool) {
		if n.Type == ir.Identifier {
			names[n.Name] = true
			return
		}
		if n.Type == ir.ArrayPattern || n.Type == ir.ObjectPattern || n.Type == ir.PatternField || n.Type == ir.RestPattern {
			for _, c := range n.Children {
				bind(c, names)
			}
		}
	}
	var collect func(ir.Node)
	collect = func(n ir.Node) {
		switch n.Type {
		case ir.Assign:
			if len(n.Children) > 0 {
				bind(n.Children[0], globals)
			}
		case ir.Interface:
			globals[n.Name] = true
			return
		case ir.Class:
			globals[n.Name] = true
			classes[n.Name] = n
			for _, c := range n.Children {
				if c.Type == ir.Assign {
					for _, v := range c.Children[1:] {
						collect(v)
					}
				} else {
					collect(c)
				}
			}
			return
		case ir.Function:
			// Parameters and their defaults are declarations in the function's scope.
			for _, c := range n.Children {
				if c.Type == ir.Block {
					collect(c)
				}
			}
			return
		case ir.Import:
			if n.DataType == "from" {
				for _, c := range n.Children {
					name := c.Name
					if c.Value != "" {
						name = c.Value
					}
					globals[name] = true
				}
			} else {
				name := n.Value
				if name == "" {
					name = strings.Split(n.Name, ".")[0]
				}
				globals[name] = true
			}
			return
		}
		for _, c := range n.Children {
			collect(c)
		}
	}
	collect(root)
	// Locally declared bases expose their fields as lexical names. An imported
	// or dynamically selected base may provide additional names; leave those
	// reads to the runtime rather than rejecting a potentially valid program.
	var fields func(ir.Node, map[string]bool, map[string]bool)
	fields = func(n ir.Node, names, seen map[string]bool) {
		if seen[n.Name] {
			return
		}
		seen[n.Name] = true
		for _, b := range n.Bases {
			if base, ok := classes[b.Name]; ok && b.Type == ir.Identifier {
				fields(base, names, seen)
			} else {
				names["*"] = true
			}
		}
		if n.Base != nil {
			if base, ok := classes[n.Base.Name]; ok && n.Base.Type == ir.Identifier {
				fields(base, names, seen)
			} else {
				names["*"] = true
			}
		}
		for _, c := range n.Children {
			if c.Type == ir.Assign && len(c.Children) > 0 {
				bind(c.Children[0], names)
			}
		}
	}
	out := []diagnostic.Diagnostic{}
	report := func(n ir.Node) {
		d := diagnostic.Diagnostic{Line: n.Line, Column: 1, EndLine: n.Line, EndColumn: 1, Code: diagnostic.UndefinedName, Severity: "error", Message: fmt.Sprintf("undefined variable %q", n.Name)}
		for _, t := range tokens {
			if t.Line == n.Line && t.Name == n.Name && (t.Type == lexer.IDENTIFIER || t.Type == lexer.SELF) {
				d.Column = t.Column
				d.EndLine = t.EndLine
				d.EndColumn = t.EndColumn
				break
			}
		}
		out = append(out, d)
	}
	copyNames := func(names map[string]bool) map[string]bool {
		copy := map[string]bool{}
		for k, v := range names {
			copy[k] = v
		}
		return copy
	}
	var walk func(ir.Node, map[string]bool)
	walk = func(n ir.Node, names map[string]bool) {
		switch n.Type {
		case ir.Identifier:
			if !names[n.Name] && !names["*"] {
				report(n)
			}
			return
		case ir.Import, ir.Interface:
			return
		case ir.Member:
			if len(n.Children) > 0 {
				walk(n.Children[0], names)
			}
			return
		case ir.Assign:
			if len(n.Children) > 0 && (n.Children[0].Type == ir.Member || n.Children[0].Type == ir.Index) {
				walk(n.Children[0], names)
			}
			for _, c := range n.Children[1:] {
				walk(c, names)
			}
			return
		case ir.Function:
			local := copyNames(names)
			// A function may subsequently be attached to an object as a method.
			local["self"] = true
			local["super"] = true
			var parameter func(ir.Node)
			parameter = func(p ir.Node) {
				switch p.Type {
				case ir.Identifier:
					local[p.Name] = true
				case ir.Tuple:
					for _, c := range p.Children {
						parameter(c)
					}
				case ir.Assign:
					if len(p.Children) > 0 {
						bind(p.Children[0], local)
					}
					for _, c := range p.Children[1:] {
						walk(c, names)
					}
				}
			}
			for _, c := range n.Children {
				if c.Type != ir.Block {
					parameter(c)
				}
			}
			for _, c := range n.Children {
				if c.Type == ir.Block {
					walk(c, local)
				}
			}
			return
		case ir.Class:
			for _, b := range n.Bases {
				walk(b, names)
			}
			for _, b := range n.Interfaces {
				walk(b, names)
			}
			if n.Base != nil {
				walk(*n.Base, names)
			}
			local := copyNames(names)
			local["self"] = true
			local["super"] = true
			fields(n, local, map[string]bool{})
			for _, c := range n.Children {
				walk(c, local)
			}
			return
		case ir.Block:
			// String-keyed object literals bind self while evaluating field values.
			if len(n.Children) > 0 && n.Children[0].Type == ir.Assign && len(n.Children[0].Children) > 0 && n.Children[0].Children[0].Type == ir.Literal {
				names = copyNames(names)
				names["self"] = true
			}
		case ir.Call, ir.If, ir.For, ir.While:
			if n.DataType == "direct-call" && !builtins[n.Name] && !names[n.Name] && !names["*"] {
				report(n)
			}
		}
		for _, c := range n.Children {
			walk(c, names)
		}
	}
	walk(root, globals)
	return out
}
