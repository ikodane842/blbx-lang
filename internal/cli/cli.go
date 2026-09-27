// Package cli implements the BLBX command line. It never exits the host process.
package cli

import (
	"blbx_lang/runtime/graph"
	"blbx_lang/runtime/graphvm"
	"blbx_lang/runtime/interpreter"
	"blbx_lang/syntax/check"
	"blbx_lang/syntax/diagnostic"
	"blbx_lang/syntax/ir"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

var Version = "0.1.0-dev"

const help = `BLBX — language tools

Usage:
  blbx check [--format text|json] [--stdin-filename name.bx] <file.bx>...
  blbx check [--format text|json] --stdin-filename name.bx -
  blbx run <file.bx> [arguments...]
  blbx run-graph <file.bx> [arguments...]
  blbx graph <file.bx>
  blbx version
  blbx help

check validates syntax and undefined names without executing code or resolving imports.
run-graph executes the same syntax using the graph backend; graph emits its JSON graph.
Use - to read source from stdin. Options must precede file paths.
Exit codes: 0 success, 1 source/runtime errors, 2 usage or I/O errors.
`

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, help)
		return 0
	}
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Fprint(stdout, help)
		return 0
	case "version", "--version":
		fmt.Fprintln(stdout, "blbx "+Version)
		return 0
	case "check":
		return runCheck(args[1:], stdin, stdout, stderr)
	case "graph":
		return dumpGraph(args[1:], stdout, stderr)
	case "run", "run-graph":
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Fprint(stdout, help)
			return 0
		}
		if len(args) < 2 {
			fmt.Fprintf(stderr, "usage: blbx %s <file.bx> [arguments...]\n", args[0])
			return 2
		}
		var err error
		if args[0] == "run-graph" {
			runtime := graphvm.New()
			runtime.SetInput(stdin)
			runtime.SetOutput(stdout)
			runtime.ErrorWriter = stderr
			runtime.Args = append([]string(nil), args[2:]...)
			_, err = runtime.ExecuteFile(args[1])
		} else {
			runtime := interpreter.New()
			runtime.SetInput(stdin)
			runtime.SetOutput(stdout)
			runtime.ErrorWriter = stderr
			runtime.Args = append([]string(nil), args[2:]...)
			_, err = runtime.ExecuteFile(args[1])
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			var pathError *os.PathError
			if errors.As(err, &pathError) {
				return 2
			}
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q; use blbx help\n", args[0])
		return 2
	}
}

func dumpGraph(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: blbx graph <file.bx>")
		return 2
	}
	if args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(stdout, "usage: blbx graph <file.bx>\nEmit a JSON execution graph without executing code or resolving imports.")
		return 0
	}
	source, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	nodes, diagnostics := check.Source(args[0], string(source))
	if len(diagnostics) != 0 {
		for _, d := range diagnostics {
			fmt.Fprintln(stderr, d.Error())
		}
		return 1
	}
	if err := graph.Build(ir.Lower(nodes), args[0]).WriteJSON(stdout); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}

func runCheck(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", "text", "diagnostic format: text or json")
	name := flags.String("stdin-filename", "<stdin>", "source name used for stdin diagnostics")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(stderr, "format must be text or json")
		return 2
	}
	if flags.NArg() == 0 {
		fmt.Fprintln(stderr, "check requires at least one file or - for stdin")
		return 2
	}
	stdinCount := 0
	for _, path := range flags.Args() {
		if path == "-" {
			stdinCount++
		}
	}
	if stdinCount > 1 {
		fmt.Fprintln(stderr, "stdin may only be read once")
		return 2
	}
	diagnostics := []diagnostic.Diagnostic{}
	exitCode := 0
	for _, path := range flags.Args() {
		var data []byte
		var err error
		if path == "-" {
			data, err = io.ReadAll(stdin)
			path = *name
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{File: path, Line: 1, Column: 1, EndLine: 1, EndColumn: 1, Severity: "error", Code: diagnostic.SourceRead, Message: err.Error()})
			exitCode = 2
			continue
		}
		_, found := check.Source(path, string(data))
		diagnostics = append(diagnostics, found...)
		if len(found) > 0 && exitCode == 0 {
			exitCode = 1
		}
	}
	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(diagnostics); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	} else {
		for _, d := range diagnostics {
			if _, err := fmt.Fprintln(stdout, d.Error()); err != nil {
				fmt.Fprintln(stderr, err)
				return 2
			}
		}
	}
	return exitCode
}
