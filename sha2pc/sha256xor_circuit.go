package sha2pc

import (
	"bytes"
	_ "embed" // Embed SHA256(XOR) circuit.

	"github.com/markkurossi/mpc/circuit"
)

// sha256xorCircuitBlob stores the compiled SHA256(XOR) circuit.
//
//go:embed sha256xor.mpclc
var sha256xorCircuitBlob []byte

// sha256xorCircuit holds the parsed circuit singleton.
var sha256xorCircuit = func() *circuit.Circuit {
	circ, err := circuit.ParseMPCLC(bytes.NewReader(sha256xorCircuitBlob))
	if err != nil {
		panic(err)
	}

	return circ
}()
