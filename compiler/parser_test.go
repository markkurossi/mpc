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

	"github.com/markkurossi/mpc/compiler/utils"
)

const (
	verbose = false
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
  return a * b + c * d
}`,
	`
package main
func main(a, b int4) (int5) {
  return a * b * c
}`,
	`
package main
func main(a, b int4) (int5) {
  return a + b + c * d + e
}`,
	`
package main
func main(a, b int4) (int4) {
  if a > b {
    return a
  }
  return b
}`,
	`
package main
func main(a, b int4) (int4) {
  if a > b {
    return a
  } else {
    return b
  }
}`,
	`
package main
func main(a, b int4) (int4) {
  if a > b || a == b {
    return a
  } else {
    return b
  }
}`,
	`
package main
func main(a, b int4) int4 {
    return max(a, b)
}

func max(a, b int4) int4 {
    if a > b {
        return a
    }
    return v
}
`,
}

func TestParser(t *testing.T) {
	min := 0
	for idx, test := range parserTests {
		if idx < min {
			continue
		}
		logger := utils.NewLogger(fmt.Sprintf("{test %d}", idx), os.Stdout)
		parser := NewParser(nil, logger, bytes.NewReader([]byte(test)))
		pkg, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse test %d failed: %v", idx, err)
		}
		if verbose {
			fmt.Printf("package %s\n", pkg.Name)
			for _, f := range pkg.Functions {
				f.Fprint(os.Stdout, 0)
				fmt.Printf("\n")
			}
		}
	}
}
