package sha2pc

import (
	"crypto/elliptic"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/ot"
)

// EvaluatorSession captures the immutable evaluator-side context between rounds.
// It is produced by EvaluatorRound2, consumed in EvaluatorRound4, and can be
// persisted and restored without additional mutation.
type EvaluatorSession struct {
	// SessionID identifies the protocol instance to catch mix-ups across runs.
	SessionID uint64

	// ChoiceBundle holds the evaluator's CO OT receiver state.
	ChoiceBundle ot.COChoiceBundle
}

// EvaluatorRound2 ingests the Round1 payload and evaluator input,
// returning the Round2 payload and updated state.
// EvaluatorRound2 requires a non-nil randomness source and curve.
func EvaluatorRound2(rng io.Reader, curve elliptic.Curve, msg Round1Payload, preimagePart [sha256.Size]byte) (Round2Payload, *EvaluatorSession, error) {
	if rng == nil {
		return Round2Payload{}, nil, errNilRandomSource
	}
	if curve == nil {
		return Round2Payload{}, nil, errNilCurve
	}
	circ := sha256xorCircuit
	if circ.NumParties() != 2 {
		return Round2Payload{}, nil, fmt.Errorf("expected 2-party circuit, got %d", circ.NumParties())
	}
	state := &EvaluatorSession{}
	if msg.OT.CurveName != curve.Params().Name {
		return Round2Payload{}, nil, fmt.Errorf("curve mismatch: %s vs %s",
			msg.OT.CurveName, curve.Params().Name)
	}
	state.SessionID = msg.SessionID
	bits := bytesToBitsLittle(preimagePart[:])
	if len(bits) != hashInputBitCount {
		return Round2Payload{}, nil, fmt.Errorf("evaluator input mismatch: got %d bits want %d",
			len(bits), hashInputBitCount)
	}

	bundle, choices, err := ot.BuildCOChoices(rng, curve, msg.OT.A.X, msg.OT.A.Y, bits)
	if err != nil {
		return Round2Payload{}, nil, err
	}
	state.ChoiceBundle = bundle

	return Round2Payload{
		SessionID: msg.SessionID,
		CurveName: curve.Params().Name,
		Choices:   choices,
	}, state, nil
}

// EvaluatorRound4 processes Round3 payload, returns the hash and Round4 payload.
// The curve argument must be non-nil.
func EvaluatorRound4(curve elliptic.Curve, state *EvaluatorSession, msg Round3Payload) ([sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	if state == nil || len(state.ChoiceBundle.Scalars) == 0 {
		return digest, fmt.Errorf("invalid evaluator state for round 4")
	}
	if curve == nil {
		return digest, errNilCurve
	}
	if msg.SessionID != state.SessionID {
		return digest, fmt.Errorf("session id mismatch: got %d want %d",
			msg.SessionID, state.SessionID)
	}
	labels, err := ot.DecryptCOCiphertexts(curve, state.ChoiceBundle, msg.Ciphertexts)
	if err != nil {
		return digest, err
	}
	totalWires := int(sha256xorCircuit.NumWires)
	wires := make([]ot.Label, totalWires)
	copy(wires[:hashInputBitCount], msg.GarblerInputs)
	copy(wires[hashInputBitCount:], labels)

	if err := sha256xorCircuit.Eval(msg.Key[:], wires, msg.GarbledTables); err != nil {
		return digest, err
	}

	if len(msg.OutputHints) != sha256xorCircuit.Outputs.Size() {
		return digest, fmt.Errorf("output hint mismatch: have %d want %d",
			len(msg.OutputHints), sha256xorCircuit.Outputs.Size())
	}

	start := sha256xorCircuit.NumWires - len(msg.OutputHints)
	outputBits := make([]bool, len(msg.OutputHints))
	for i := 0; i < len(msg.OutputHints); i++ {
		bit, err := circuit.BitFromLabel(msg.OutputHints[i], wires[start+i])
		if err != nil {
			return digest, err
		}
		outputBits[i] = bit
	}

	bytes := bitsToBytesLittle(outputBits)
	if len(bytes) != sha256.Size {
		return digest,
			fmt.Errorf("unexpected output length %d", len(bytes))
	}

	copy(digest[:], bytes)

	return digest, nil
}
