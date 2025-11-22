package sha2pc

// bytesToBitsLittle converts a byte slice into a little-endian bit slice.
func bytesToBitsLittle(data []byte) []bool {
	bits := make([]bool, len(data)*8)
	for idx, b := range data {
		for bit := 0; bit < 8; bit++ {
			if b&(1<<uint(bit)) != 0 {
				bits[idx*8+bit] = true
			}
		}
	}

	return bits
}

// bitsToBytesLittle packs a bit slice back into bytes using little-endian order.
func bitsToBytesLittle(bits []bool) []byte {
	size := (len(bits) + 7) / 8
	result := make([]byte, size)
	for idx, bit := range bits {
		if bit {
			byteIdx := idx / 8
			result[byteIdx] |= 1 << uint(idx%8)
		}
	}

	return result
}
