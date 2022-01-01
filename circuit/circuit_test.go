//
// Copyright (c) 2022 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"testing"
	"unsafe"
)

func TestSize(t *testing.T) {
	var g Gate
	fmt.Printf("sizeof(Gate)=%d\n", unsafe.Sizeof(g))
}
