package sha2pc

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/ot"
)

// errNilRandomSource indicates that a required randomness source was nil.
var errNilRandomSource = errors.New("sha2pc: randomness source must not be nil")

// errNilCurve indicates that a required elliptic curve was nil.
var errNilCurve = errors.New("sha2pc: elliptic curve must not be nil")

// GarblerSession captures the immutable garbler-side context between rounds.
// It is produced by GarblerRound1, consumed in GarblerRound3, and can be
// persisted and restored without additional mutation.
type GarblerSession struct {
	// SessionID identifies the protocol instance to catch mix-ups across runs.
	SessionID uint64

	// SenderSetup is the immutable CO OT sender configuration.
	SenderSetup ot.COSenderSetup
}

// GarblerRound1 sets up the OT sender side and returns the outbound payload plus
// the immutable session needed for later rounds.
// GarblerRound1 requires a non-nil randomness source and curve.
func GarblerRound1(rng io.Reader, curve elliptic.Curve) (Round1Payload, *GarblerSession, error) {
	if rng == nil {
		return Round1Payload{}, nil, errNilRandomSource
	}
	if curve == nil {
		return Round1Payload{}, nil, errNilCurve
	}

	setup, err := ot.GenerateCOSenderSetup(rng, curve)
	if err != nil {
		return Round1Payload{}, nil, err
	}

	var sidBuf [8]byte
	if _, err := io.ReadFull(rng, sidBuf[:]); err != nil {
		return Round1Payload{}, nil, fmt.Errorf("failed to read session id: %w", err)
	}
	sessionID := binary.BigEndian.Uint64(sidBuf[:])

	state := &GarblerSession{
		SessionID:   sessionID,
		SenderSetup: setup,
	}

	payload := Round1Payload{
		SessionID: sessionID,
		OT: OTSenderSetup{
			CurveName: setup.CurveName,
			A: ot.ECPoint{
				X: new(big.Int).Set(setup.Ax),
				Y: new(big.Int).Set(setup.Ay),
			},
		},
	}

	return payload, state, nil
}

// GarblerRound3 garbles the circuit for the provided input, encrypts the OT
// payloads, and emits the message for the evaluator. Both rng and curve must
// be non-nil.
func GarblerRound3(rng io.Reader, curve elliptic.Curve, state *GarblerSession, preimagePart [sha256.Size]byte, req Round2Payload) (Round3Payload, error) {
	if rng == nil {
		return Round3Payload{}, errNilRandomSource
	}
	if state == nil || state.SenderSetup.Scalar == nil {
		return Round3Payload{}, errors.New("sha2pc: invalid garbler session")
	}
	if curve == nil {
		return Round3Payload{}, errNilCurve
	}
	circ := sha256xorCircuit
	if circ.NumParties() != 2 {
		return Round3Payload{}, fmt.Errorf("expected 2-party circuit, got %d", circ.NumParties())
	}
	if req.SessionID != state.SessionID {
		return Round3Payload{}, fmt.Errorf("session id mismatch: got %d want %d",
			req.SessionID, state.SessionID)
	}

	var key [32]byte
	if _, err := io.ReadFull(rng, key[:]); err != nil {
		return Round3Payload{}, fmt.Errorf("failed to read garbling key: %w", err)
	}

	garbled, err := circ.Garble(rng, key[:])
	if err != nil {
		return Round3Payload{}, err
	}

	gBits := hashInputBitCount
	eBits := hashInputBitCount
	bits := bytesToBitsLittle(preimagePart[:])
	if len(bits) != gBits {
		return Round3Payload{}, fmt.Errorf("garbler input mismatch: got %d bits want %d",
			len(bits), gBits)
	}
	garblerLabels := make([]ot.Label, gBits)
	for i := 0; i < gBits; i++ {
		garblerLabels[i] = circuit.LabelForBit(garbled.Wires[i], bits[i])
	}
	evaluatorWires := garbled.Wires[gBits : gBits+eBits]
	ciphertexts, err := ot.EncryptCOCiphertexts(curve, state.SenderSetup, req.Choices, evaluatorWires)
	if err != nil {
		return Round3Payload{}, err
	}

	outputs := circ.Outputs.Size()
	outputHints := make([]ot.Wire, outputs)
	start := int(circ.NumWires) - outputs
	copy(outputHints, garbled.Wires[start:])

	return Round3Payload{
		SessionID:     state.SessionID,
		Ciphertexts:   ciphertexts,
		Key:           key,
		GarbledTables: garbled.Gates,
		GarblerInputs: garblerLabels,
		OutputHints:   outputHints,
	}, nil
}
