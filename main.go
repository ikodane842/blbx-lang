package main

import (
	"blbx_lang/syntax/lexer"
	"fmt"
	"os"
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

	// print tokens
	for _, tok := range l.Get() {
		fmt.Printf("Line %-3d | %-10s | %q\n",
			tok.Line,
			tok.Type, // will print nicely if you added String() to TokenType
			tok.Name,
		)
	}
}
