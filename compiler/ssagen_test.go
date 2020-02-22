//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"
	"testing"
)

type SSAGenTest struct {
	Enabled bool
	Name    string
	Code    string
}

var ssagenTests = []SSAGenTest{
	SSAGenTest{
		Enabled: true,
		Name:    "add",
		Code: `
package main
func main(a, b int) int {
    return a + b
}
`,
	},
	SSAGenTest{
		Enabled: true,
		Name:    "add3",
		Code: `
package main
func main(a, b, e int) (int) {
    return a + b + e
}
`,
	},
	SSAGenTest{
		Enabled: false,
		Name:    "ret2",
		Code: `
package main
func main(a, b int) (int, int) {
    return a + b, a - b
}
`,
	},
	SSAGenTest{
		Enabled: false,
		Name:    "if",
		Code: `
package main
func main(a, b int) int {
    if a > b {
        return a
    }
    return b
}
`,
	},
	SSAGenTest{
		Enabled: false,
		Name:    "ifelse",
		Code: `
package main
func main(a, b int) int {
    if a > b {
        return a
    } else {
        return b
    }
}
`,
	},
	SSAGenTest{
		Enabled: false,
		Name:    "if-else-assign",
		Code: `
package main
func main(a, b int) (int, int) {
    var max, min int
    if a > b {
        max = a
        min = b
    } else {
        max = b
        min = a
    }
    min = min + max
    max = max + min
    return min, max
}
`,
	},
	SSAGenTest{
		Enabled: false,
		Name:    "max3",
		Code: `
package main
func main(a, b, c int) int {
    var max int
    if a > b {
        if a > c {
            max = a
        } else {
            max = c
        }
    } else {
        if b > c {
            max = b
        } else {
            max = c
        }
    }
    return max
}
`,
	},
	SSAGenTest{
		Enabled: true,
		Name:    "Millionaire",
		Code: `
package main
func main(a, b int) int {
    if a > b {
        return 0
    } else {
        return 1
    }
}
`,
	},
	SSAGenTest{
		Enabled: true,
		Name:    "Mult",
		Code: `
package main
func main(a, b int) int {
    return a * b
}
`,
	},
	SSAGenTest{
		Enabled: true,
		Name:    "Bool",
		Code: `
package main
func main(a, b int) bool {
    if a > b {
        return true
    }
    return false
}
`,
	},
}

func TestSSAGen(t *testing.T) {
	for idx, test := range ssagenTests {
		if !test.Enabled {
			continue
		}
		if false {
			fmt.Printf(`==================================================
// Test '%s':
%s--------------------------------------------------
`,
				test.Name, test.Code)
		}
		_, err := Compile(test.Code, &Params{})
		if err != nil {
			t.Fatalf("SSA test %d failed: %s", idx, err)
		}
	}
}
