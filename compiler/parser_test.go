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
	"os"
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
  return
}`,
	`
package main
func main(a, b int4) (int5) {
  return a + b
}`,
}

func TestParser(t *testing.T) {
	min := 0
	for idx, test := range parserTests {
		if idx < min {
			continue
		}
		parser := NewParser(fmt.Sprintf("{test %d}", idx),
			bytes.NewReader([]byte(test)))
		unit, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		fmt.Printf("package %s\n", unit.Package)
		if unit.AST != nil {
			unit.AST.Fprint(os.Stdout, 0)
			fmt.Printf("\n")
		}
	}
}
