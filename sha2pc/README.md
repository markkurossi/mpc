# sha2pc

`sha2pc` is a pure-Go two-party protocol that privately computes
`SHA256(XOR(a, b))` for two 32-byte inputs. The package exposes
round handlers that return explicit Go structs, so callers can exchange
the typed payloads (`Round1Payload`, `Round2Payload`, etc.) using any
transport they like (sockets, gRPC, files, etc.).

## Example

```go
package main

import (
	crand "crypto/rand"
	"fmt"
	"log"

	"github.com/markkurossi/mpc/sha2pc"
)

func main() {
	var a, b [32]byte
	for i := 0; i < len(a); i++ {
		a[i] = byte(i)
		b[i] = byte(len(a) - i)
	}

	msg1, gState, err := sha2pc.GarblerRound1(crand.Reader, sha2pc.CurveP256)
	if err != nil {
		log.Fatalf("round1: %v", err)
	}
	fmt.Printf("Round1: curve=%s\n", msg1.OT.CurveName)

	msg2, eState, err := sha2pc.EvaluatorRound2(crand.Reader, sha2pc.CurveP256, msg1, b)
	if err != nil {
		log.Fatalf("round2: %v", err)
	}
	fmt.Printf("Round2: choices=%d\n", len(msg2.Choices))

	msg3, err := sha2pc.GarblerRound3(crand.Reader, sha2pc.CurveP256, gState, a, msg2)
	if err != nil {
		log.Fatalf("round3: %v", err)
	}
	fmt.Printf("Round3: ciphertexts=%d tables=%d\n", len(msg3.Ciphertexts), len(msg3.GarbledTables))

	hashEval, err := sha2pc.EvaluatorRound4(sha2pc.CurveP256, eState, msg3)
	if err != nil {
		log.Fatalf("round4: %v", err)
	}

	fmt.Printf("Evaluator hash: %x\n", hashEval)
}
```
