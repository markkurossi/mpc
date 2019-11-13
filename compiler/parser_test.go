//
// parser_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"bytes"
	"fmt"
	"testing"
)

var parserTests = []string{
	`
package main
`,
	`
package main
func main() {}
`,
	`
package main
func main(a int4) {}
`,
	`
package main
func main(a int4, b int4) {}
`,
	`
package main
func main(a, b int4) {}
`,
	`
package main
func main(a, b int4) int5 {}
`,
	`
package main
func main(a, b int4) (int5) {}
`,
	`
package main
func main(a, b int4) (int5, int6) {}
`,
	`
package main
func main(a, b int4) (int5) {
  return a + b
}`,
}

func TestParser(t *testing.T) {
	for idx, test := range parserTests {
		parser := NewParser(fmt.Sprintf("{test %d}", idx),
			bytes.NewReader([]byte(test)))
		_, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
	}
}
