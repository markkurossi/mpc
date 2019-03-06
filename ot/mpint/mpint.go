//
// mpint.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package mpint

import (
	"math/big"
)

func FromBytes(data []byte) *big.Int {
	return big.NewInt(0).SetBytes(data)
}

func Add(a, b *big.Int) *big.Int {
	return big.NewInt(0).Add(a, b)
}

func Sub(a, b *big.Int) *big.Int {
	return big.NewInt(0).Sub(a, b)
}

func Exp(x, y, m *big.Int) *big.Int {
	return big.NewInt(0).Exp(x, y, m)
}

func Mod(x, y *big.Int) *big.Int {
	return big.NewInt(0).Mod(x, y)
}
