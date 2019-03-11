//
// parser_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"bytes"
	"fmt"
	"testing"
)

var data = `375 439
32 32   33

2 1 0 32 406 XOR
2 1 5 37 373 AND
2 1 4 36 336 AND
2 1 10 42 340 AND
2 1 14 46 366 AND
2 1 24 56 341 AND
2 1 8 40 342 AND
`

func TestParse(t *testing.T) {
	circuit, err := Parse(bytes.NewReader([]byte(data)))
	if err != nil {
		t.Fatalf("Parse failed: %s", err)
	}
	fmt.Printf("Circuit: %#v\n", circuit)
}
