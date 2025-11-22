package sha2pc

import "github.com/markkurossi/mpc/ot"

// Round1Payload carries everything the garbler sends in round 1.
type Round1Payload struct {
	// SessionID ties all rounds together to catch accidental mixing.
	SessionID uint64

	// OT contains the OT sender setup information.
	OT OTSenderSetup
}

// OTSenderSetup exposes the sender metadata needed for OT.
type OTSenderSetup struct {
	// CurveName identifies the elliptic curve (e.g., "P-256").
	CurveName string

	// A stores the sender's public curve point.
	A ot.ECPoint
}

// Round2Payload carries the evaluator's OT choices back to the garbler.
type Round2Payload struct {
	// SessionID echoes the negotiated session identifier.
	SessionID uint64

	// CurveName identifies the elliptic curve used for OT.
	CurveName string

	// Choices stores one curve point per input bit.
	Choices []ot.ECPoint
}

// Round3Payload carries OT ciphertexts back to the evaluator.
type Round3Payload struct {
	// SessionID echoes the negotiated session identifier.
	SessionID uint64

	// Ciphertexts contains two encrypted labels per input bit.
	Ciphertexts []ot.LabelCiphertext

	// Key is the 32-byte AES garbling key.
	Key [32]byte

	// GarbledTables holds every gate's ciphertext row.
	GarbledTables [][]ot.Label

	// GarblerInputs contains the garbler's input wire labels.
	GarblerInputs []ot.Label

	// OutputHints lists both labels for each output wire.
	OutputHints []ot.Wire
}
