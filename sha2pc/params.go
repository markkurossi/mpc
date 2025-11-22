package sha2pc

import (
	"crypto/elliptic"
	"fmt"

	"github.com/markkurossi/mpc/circuit"
)

// CurveP256 is the default curve used for SHA256(XOR) evaluations.
var CurveP256 = elliptic.P256()

const (
	sessionIDBytes = 8

	// hashInputBitCount locks the 32-byte preimage size (256 bits).
	hashInputBitCount = 32 * 8

	labelByteLen = 16

	// garblingKeyBytes is the size of the AES garbling key per round.
	garblingKeyBytes = 32

	// garbledTableLabelCount is the total number of ciphertext labels emitted by
	// the SHA256(XOR) garbled circuit (derived from its AND/OR/INV gate counts).
	garbledTableLabelCount = 42914

	// garbledTableByteLen is the total byte length of all garbled table labels.
	garbledTableByteLen = garbledTableLabelCount * labelByteLen

	// garblerInputLabelCount captures how many labels the garbler sends in round 3.
	garblerInputLabelCount = hashInputBitCount

	garblerInputLabelBytes = garblerInputLabelCount * labelByteLen

	// evaluatorCiphertextCount equals the number of evaluator input bits.
	evaluatorCiphertextCount = hashInputBitCount

	ciphertextBytes = evaluatorCiphertextCount * 2 * labelByteLen

	// outputHintCount equals the SHA256 output bits.
	outputHintCount = 256

	outputHintBytes = outputHintCount * 2 * labelByteLen

	// evaluatorChoiceSignBytes is the number of bytes needed for compressed point signs.
	evaluatorChoiceSignBytes = (evaluatorCiphertextCount + 7) / 8

	// round3PayloadLen is the fixed number of bytes in a Round 3 payload.
	round3PayloadLen = len(magicRound3) + sessionIDBytes + garblingKeyBytes +
		garbledTableByteLen + garblerInputLabelBytes + outputHintBytes + ciphertextBytes
)

// init validates that the circuit matches the expected consts.
func init() {
	if bits := int(sha256xorCircuit.Inputs[0].Type.Bits); bits != hashInputBitCount {
		panic(fmt.Sprintf("garbler bit-count mismatch: %d != %d", bits, hashInputBitCount))
	}
	if bits := int(sha256xorCircuit.Inputs[1].Type.Bits); bits != hashInputBitCount {
		panic(fmt.Sprintf("evaluator bit-count mismatch: %d != %d", bits, hashInputBitCount))
	}

	var labels int
	for _, gate := range sha256xorCircuit.Gates {
		count, err := gateCiphertextCount(gate.Op)
		if err != nil {
			panic(err)
		}
		labels += count
	}
	if labels != garbledTableLabelCount {
		panic(fmt.Sprintf("garbled table label mismatch: %d != %d", labels, garbledTableLabelCount))
	}

	if outputs := sha256xorCircuit.Outputs.Size(); outputs != outputHintCount {
		panic(fmt.Sprintf("output hint count mismatch: %d != %d", outputs, outputHintCount))
	}
}

func gateCiphertextCount(op circuit.Operation) (int, error) {
	switch op {
	case circuit.XOR, circuit.XNOR:
		return 0, nil
	case circuit.AND:
		return 2, nil
	case circuit.OR:
		return 3, nil
	case circuit.INV:
		return 1, nil
	default:
		return 0, fmt.Errorf("sha2pc: unsupported gate operation %v", op)
	}
}
