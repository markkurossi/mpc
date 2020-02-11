//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"testing"
)

type SSAGenTest struct {
	Code string
}

var ssagenTests = []SSAGenTest{
	SSAGenTest{
		Code: `
package main
func main(a, b int) int {
    return a + b
}
`,
	},
	SSAGenTest{
		Code: `
package main
func main(a, b int) (int, int) {
    return a + b, a - b
}
`,
	},
}

func TestSSAGen(t *testing.T) {
	for idx, test := range ssagenTests {
		_, err := Compile(test.Code)
		if err != nil {
			t.Fatalf("SSA test %d failed: %s", idx, err)
		}
	}
}
