package sha2pc

import (
	crand "crypto/rand"
	"fmt"
)

// Example demonstrates running every round and inspecting the payloads.
func Example() {
	var a, b [32]byte
	for i := 0; i < len(a); i++ {
		a[i] = byte(i)
		b[i] = byte(len(a) - i)
	}

	msg1, gState, err := GarblerRound1(crand.Reader, CurveP256)
	if err != nil {
		panic(err)
	}
	fmt.Printf("round1 curve=%s\n", msg1.OT.CurveName)
	r1Bytes, err := EncodeRound1(CurveP256, msg1)
	if err != nil {
		panic(err)
	}
	gSessionBytes, err := EncodeGarblerSession(CurveP256, gState)
	if err != nil {
		panic(err)
	}
	fmt.Printf("round1 encode=%d session=%d\n", len(r1Bytes), len(gSessionBytes))

	msg2, eState, err := EvaluatorRound2(crand.Reader, CurveP256, msg1, b)
	if err != nil {
		panic(err)
	}
	fmt.Printf("round2 choices=%d\n", len(msg2.Choices))
	r2Bytes, err := EncodeRound2(CurveP256, msg2)
	if err != nil {
		panic(err)
	}
	eSessionBytes, err := EncodeEvaluatorSession(CurveP256, eState)
	if err != nil {
		panic(err)
	}
	fmt.Printf("round2 encode=%d session=%d\n", len(r2Bytes), len(eSessionBytes))

	msg3, err := GarblerRound3(crand.Reader, CurveP256, gState, a, msg2)
	if err != nil {
		panic(err)
	}
	fmt.Printf("round3 ciphertexts=%d tables=%d\n",
		len(msg3.Ciphertexts), len(msg3.GarbledTables))
	r3Bytes, err := EncodeRound3(msg3)
	if err != nil {
		panic(err)
	}
	fmt.Printf("round3 encode=%d\n", len(r3Bytes))

	hashEval, err := EvaluatorRound4(CurveP256, eState, msg3)
	if err != nil {
		panic(err)
	}
	fmt.Printf("round4 digest-prefix=%x\n", hashEval[:4])

	fmt.Printf("evaluator hash=%x\n", hashEval)

	// Output:
	// round1 curve=P-256
	// round1 encode=80 session=178
	// round2 choices=256
	// round2 encode=8240 session=8306
	// round3 ciphertexts=256 tables=127806
	// round3 encode=707146
	// round4 digest-prefix=4b2f7457
	// evaluator hash=4b2f74579fc7c778745121996f604371a326dc5174f9851706032626668abf2e
}
