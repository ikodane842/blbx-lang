// Black Box uses the existing BLBX frontend with the graph execution backend.
package main

import (
	"blbx_lang/internal/cli"
	"fmt"
	"os"
)

func main() {
	args := append([]string(nil), os.Args[1:]...)
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" ||
		(len(args) == 2 && args[0] == "run" && (args[1] == "--help" || args[1] == "-h")) {
		fmt.Print(`Black Box — BLBX syntax, graph execution

Usage:
  blackbox run <file.bx> [arguments...]
  blackbox graph <file.bx>
  blackbox check [--format text|json] <file.bx>...
  blackbox version

run traverses the execution graph. Explicit calls execute even when their
return values are unused. graph writes JSON without executing the script.
Pure scalar values may be shared; state changes and other effects stay ordered.
check accepts the same options as blbx check.
`)
		return
	}
	if args[0] == "version" || args[0] == "--version" {
		fmt.Println("blackbox " + cli.Version + " (graph backend)")
		return
	}
	if args[0] == "run" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: blackbox run <file.bx> [arguments...]")
			os.Exit(2)
		}
		args[0] = "run-graph"
	}
	os.Exit(cli.Run(args, os.Stdin, os.Stdout, os.Stderr))
}
