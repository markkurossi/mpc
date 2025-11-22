package sha2pc

import "testing"

// TestBytesToBitsLittleRoundTrip verifies bytesToBitsLittle is invertible
// through bitsToBytesLittle.
func TestBytesToBitsLittleRoundTrip(t *testing.T) {
	data := []byte{0b10110010, 0b01101100}
	bits := bytesToBitsLittle(data)
	got := bitsToBytesLittle(bits)
	if len(got) != len(data) {
		t.Fatalf("length mismatch: got %d want %d", len(got), len(data))
	}
	for i := range data {
		if got[i] != data[i] {
			t.Fatalf("byte %d mismatch: got %08b want %08b", i, got[i], data[i])
		}
	}
}

// TestBitsToBytesLittlePartial verifies partial bit packing.
func TestBitsToBytesLittlePartial(t *testing.T) {
	bits := []bool{true, false, true, true, false}
	bytes := bitsToBytesLittle(bits)
	if len(bytes) != 1 {
		t.Fatalf("expected 1 byte, got %d", len(bytes))
	}
	expected := byte(0)
	for idx, bit := range bits {
		if bit {
			expected |= 1 << uint(idx)
		}
	}
	if bytes[0] != expected {
		t.Fatalf("byte mismatch: got %08b want %08b", bytes[0], expected)
	}
}
