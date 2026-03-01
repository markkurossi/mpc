//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package gmw

func copyOf(bitvec []uint64) []uint64 {
	result := make([]uint64, len(bitvec))
	copy(result, bitvec)
	return result
}

func bit(bitvec []uint64, i int) uint {
	w := i / 64
	o := i % 64

	if i < 0 {
		panic("negative bit index")
	}
	if w >= len(bitvec) {
		return 0
	}
	return uint(bitvec[w]>>o) & 1
}

func setBit(bitvec []uint64, i int, b uint) []uint64 {
	w := i / 64
	o := i % 64

	if i < 0 {
		panic("negative bit index")
	}
	if w >= len(bitvec) {
		n := make([]uint64, w+1)
		copy(n, bitvec)
		bitvec = n
	}

	switch b {
	case 0:
		bitvec[w] = bitvec[w] &^ (1 << o)
	case 1:
		bitvec[w] = bitvec[w] | (1 << o)
	default:
		panic("invalid bit value")
	}

	return bitvec
}

func xorBitvec(result, bitvec []uint64) {
	for i, b := range bitvec {
		result[i] ^= b
	}
}

func expand(bitvec []uint64, words int) []uint64 {
	if words <= len(bitvec) {
		return bitvec
	}
	result := make([]uint64, words)
	copy(result, bitvec)

	return result
}
