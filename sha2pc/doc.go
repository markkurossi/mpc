// Package sha2pc implements a self-contained two-party protocol for
// computing SHA256(XOR(a,b)) without embedding any networking code.
// Each party drives the protocol by invoking the exported round
// methods and exchanging the strongly-typed payload structs with its
// peer. The package hides all MPC internal details (garbled circuits,
// oblivious transfers, etc.) while keeping the message ordering explicit.
//
// Example:
//
//	var a, b [32]byte
//	for i := 0; i < len(a); i++ {
//		a[i] = byte(i)
//		b[i] = byte(len(a) - i)
//	}
//
//	msg1, garblerState, err := sha2pc.GarblerRound1(crand.Reader, sha2pc.CurveP256)
//	if err != nil {
//		log.Fatal(err)
//	}
//	msg2, evaluatorState, err := sha2pc.EvaluatorRound2(crand.Reader, sha2pc.CurveP256, msg1, b)
//	if err != nil {
//		log.Fatal(err)
//	}
//	msg3, err := sha2pc.GarblerRound3(crand.Reader, sha2pc.CurveP256, garblerState, a, msg2)
//	if err != nil {
//		log.Fatal(err)
//	}
//	hashEval, err := sha2pc.EvaluatorRound4(sha2pc.CurveP256, evaluatorState, msg3)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Variables a and b are parts of the preimage. Variable hashEval is the final
// SHA-256 hash of XOR(a, b).
package sha2pc
