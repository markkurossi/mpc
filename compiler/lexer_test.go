//
// lexer_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

var input = `// This is a very basic add circuit.
// But we start this example with 2 comment lines.
/a/
func main(a, b int4) int5 {
    return a + b
}
`

func TestLexer(t *testing.T) {
	lexer := NewLexer(bytes.NewReader([]byte(input)))
	for {
		token, err := lexer.Get()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Get failed: %v", err)
		}
		fmt.Printf("Token: %s\n", token)
	}
}
