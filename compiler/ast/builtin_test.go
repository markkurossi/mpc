//
// Copyright (c) 2019-2025 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"
	"math"
	"math/big"
	"testing"
)

func TestBase10Digits(t *testing.T) {
	minusOne := big.NewInt(-1)

	for i := 1; i < 2048; i++ {
		ival := big.NewInt(1)

		max := new(big.Int).Lsh(ival, uint(i))
		max = max.Add(max, minusOne)

		digits := int(float64(i)*math.Log10(2)) + 1

		str := fmt.Sprintf("%v", max)

		if digits != len(str) {
			t.Errorf("%v:\t%v[%d]\t%d\n", i, str, len(str), digits)
		}
	}
}

func TestItoa(t *testing.T) {
	digits := int(float64(32)*math.Log10(2)) + 1

	result := make([]byte, digits)
	val := 987654321

	mask := 1

	for i := 0; i < digits-1; i++ {
		mask *= 10
	}

	for i := 0; i < digits; i++ {
		d := val / mask
		fmt.Printf("d=%v, val=%v, mask=%v\n", d, val, mask)

		result[i] = byte('0' + d)
		val -= d * mask
		mask /= 10
	}

	fmt.Printf("result: %v\n", string(result[:]))
}
