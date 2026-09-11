package main

import (
	"blbx_lang/syntax/lexer"
	"blbx_lang/syntax/parser"
	"encoding/json"
	"os"
	"path/filepath"
)

func main() {
	// read test file
	data, err := os.ReadFile("tests/first_test.bx")
	if err != nil {
		panic(err)
	}

	// create lexer
	var l lexer.Lexer
	l.Set(string(data))

	// run tokenizer
	l.Tokenize()

	// create parser

	var p parser.Parser
	p.Set(l.Get())

	// run parser
	p.Parse()

	writeJSON(filepath.Join("tests", "output", "first_test_output.json"), p.Get())
}

func writeJSON(outputPath string, value interface{}) {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		panic(err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	encoder := json.NewEncoder(outputFile)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(value); err != nil {
		panic(err)
	}
}
