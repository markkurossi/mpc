package sha2pc

import (
	crand "crypto/rand"
	"testing"
)

// BenchmarkProtocol exercises all rounds to measure total latency.
func BenchmarkProtocol(b *testing.B) {
	var aIn, bIn [32]byte
	for i := 0; i < len(aIn); i++ {
		aIn[i] = byte(i)
		bIn[i] = byte(len(aIn) - i)
	}

	for b.Loop() {
		msg1, gState, err := GarblerRound1(crand.Reader, CurveP256)
		if err != nil {
			b.Fatalf("Round1: %v", err)
		}
		msg2, eState, err := EvaluatorRound2(crand.Reader, CurveP256, msg1, bIn)
		if err != nil {
			b.Fatalf("Round2: %v", err)
		}
		msg3, err := GarblerRound3(crand.Reader, CurveP256, gState, aIn, msg2)
		if err != nil {
			b.Fatalf("Round3: %v", err)
		}
		if _, err := EvaluatorRound4(CurveP256, eState, msg3); err != nil {
			b.Fatalf("Round4: %v", err)
		}
	}
}
