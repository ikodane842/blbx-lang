package main

import (
	"blbx_lang/internal/cli"
	"os"
)

func main() { os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
