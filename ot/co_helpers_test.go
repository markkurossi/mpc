package ot

import (
	"bytes"
	"crypto/elliptic"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"testing"
)

// helperTranscriptHash locks the helper transcript digest.
const helperTranscriptHash = "c3abf5ccf4268d0b4a6f2607666f78df30463f2211fab258599d529ddf869779"

// TestCOHelpersDeterministicTranscript locks the pure helper pipeline to a hash.
func TestCOHelpersDeterministicTranscript(t *testing.T) {
	curve := elliptic.P256()
	senderRand := newDeterministicReader("helpers-sender")
	receiverRand := newDeterministicReader("helpers-receiver")
	wireRand := newDeterministicReader("helpers-wires")

	wires := make([]Wire, 4)
	for i := range wires {
		l0, err := readLabelFromReader(wireRand)
		if err != nil {
			t.Fatalf("read l0: %v", err)
		}
		l1, err := readLabelFromReader(wireRand)
		if err != nil {
			t.Fatalf("read l1: %v", err)
		}
		wires[i].L0 = l0
		wires[i].L1 = l1
	}
	choices := []bool{false, true, true, false}

	setup, err := GenerateCOSenderSetup(senderRand, curve)
	if err != nil {
		t.Fatalf("GenerateCOSenderSetup: %v", err)
	}
	bundle, points, err := BuildCOChoices(receiverRand, curve, setup.Ax, setup.Ay, choices)
	if err != nil {
		t.Fatalf("BuildCOChoices: %v", err)
	}
	ciphertexts, err := EncryptCOCiphertexts(curve, setup, points, wires)
	if err != nil {
		t.Fatalf("EncryptCOCiphertexts: %v", err)
	}
	labels, err := DecryptCOCiphertexts(curve, bundle, ciphertexts)
	if err != nil {
		t.Fatalf("DecryptCOCiphertexts: %v", err)
	}
	for idx, bit := range choices {
		expected := wires[idx].L0
		if bit {
			expected = wires[idx].L1
		}
		if !labels[idx].Equal(expected) {
			t.Fatalf("label %d mismatch", idx)
		}
	}

	var buf bytes.Buffer
	buf.Write(setup.Ax.Bytes())
	buf.Write(setup.Ay.Bytes())
	for _, ct := range ciphertexts {
		buf.Write(ct.Zero[:])
		buf.Write(ct.One[:])
	}
	var tmp LabelData
	for _, lbl := range labels {
		lbl.GetData(&tmp)
		buf.Write(tmp[:])
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))
	if hash != helperTranscriptHash {
		t.Fatalf("transcript hash mismatch: got %s want %s", hash, helperTranscriptHash)
	}
}

// TestCOHelpersNilCurve verifies that helpers reject nil curves.
func TestCOHelpersNilCurve(t *testing.T) {
	if _, err := GenerateCOSenderSetup(newDeterministicReader("nil-sender"), nil); err != ErrNilCurve {
		t.Fatalf("unexpected error for nil curve: %v", err)
	}
	if _, _, err := BuildCOChoices(newDeterministicReader("nil-receiver"), nil, nil, nil, nil); err != ErrNilCurve {
		t.Fatalf("unexpected error for nil curve: %v", err)
	}
	if _, err := EncryptCOCiphertexts(nil, COSenderSetup{}, nil, nil); err != ErrNilCurve {
		t.Fatalf("unexpected error for nil curve: %v", err)
	}
	if _, err := DecryptCOCiphertexts(nil, COChoiceBundle{}, nil); err != ErrNilCurve {
		t.Fatalf("unexpected error for nil curve: %v", err)
	}
}

// TestBuildCOChoicesRejectsInvalidSenderPoint ensures malicious inputs fail fast.
func TestBuildCOChoicesRejectsInvalidSenderPoint(t *testing.T) {
	curve := elliptic.P256()
	setup, err := GenerateCOSenderSetup(newDeterministicReader("invalid-curve-garbler"), curve)
	if err != nil {
		t.Fatalf("GenerateCOSenderSetup: %v", err)
	}
	fakeAy := new(big.Int).Add(setup.Ay, big.NewInt(1))
	for curve.IsOnCurve(setup.Ax, fakeAy) {
		fakeAy.Add(fakeAy, big.NewInt(1))
	}
	choices := []bool{false, true, false}
	if _, _, err := BuildCOChoices(newDeterministicReader("invalid-curve-evaluator"), curve, setup.Ax, fakeAy, choices); !errors.Is(err, ErrPointNotOnCurve) {
		t.Fatalf("BuildCOChoices error mismatch: %v", err)
	}
}

// TestEncryptCOCiphertextsRejectsInvalidChoicePoint ensures evaluator points are validated.
func TestEncryptCOCiphertextsRejectsInvalidChoicePoint(t *testing.T) {
	curve := elliptic.P256()
	setup, err := GenerateCOSenderSetup(newDeterministicReader("invalid-choice-garbler"), curve)
	if err != nil {
		t.Fatalf("GenerateCOSenderSetup: %v", err)
	}
	wires := make([]Wire, 2)
	wireRand := newDeterministicReader("invalid-choice-wires")
	for i := range wires {
		l0, err := readLabelFromReader(wireRand)
		if err != nil {
			t.Fatalf("l0: %v", err)
		}
		l1, err := readLabelFromReader(wireRand)
		if err != nil {
			t.Fatalf("l1: %v", err)
		}
		wires[i].L0 = l0
		wires[i].L1 = l1
	}
	points := make([]ECPoint, len(wires))
	for i := range points {
		x := new(big.Int).Add(setup.Ax, big.NewInt(int64(i+1)))
		y := new(big.Int).Add(setup.Ay, big.NewInt(int64(2*i+3)))
		for curve.IsOnCurve(x, y) {
			y.Add(y, big.NewInt(1))
		}
		points[i] = ECPoint{X: x, Y: y}
	}
	if _, err := EncryptCOCiphertexts(curve, setup, points, wires); !errors.Is(err, ErrPointNotOnCurve) {
		t.Fatalf("EncryptCOCiphertexts error mismatch: %v", err)
	}
}

// TestEnsureOnCurve validates that the helper enforces curve membership.
func TestEnsureOnCurve(t *testing.T) {
	curve := elliptic.P256()
	x := curve.Params().Gx
	y := curve.Params().Gy
	if err := ensureOnCurve(curve, x, y); err != nil {
		t.Fatalf("ensureOnCurve valid point: %v", err)
	}
	if err := ensureOnCurve(nil, x, y); !errors.Is(err, ErrNilCurve) {
		t.Fatalf("ensureOnCurve nil curve mismatch: %v", err)
	}
	badY := new(big.Int).Add(y, big.NewInt(1))
	for curve.IsOnCurve(x, badY) {
		badY.Add(badY, big.NewInt(1))
	}
	if err := ensureOnCurve(curve, x, badY); !errors.Is(err, ErrPointNotOnCurve) {
		t.Fatalf("ensureOnCurve invalid point mismatch: %v", err)
	}
}

// BenchmarkGenerateCOSenderSetup measures helper performance.
func BenchmarkGenerateCOSenderSetup(b *testing.B) {
	curve := elliptic.P256()
	rng := newDeterministicReader("bench-sender")
	for b.Loop() {
		if _, err := GenerateCOSenderSetup(rng, curve); err != nil {
			b.Fatalf("GenerateCOSenderSetup: %v", err)
		}
	}
}

// BenchmarkEncryptCOCiphertexts measures encryption throughput.
func BenchmarkEncryptCOCiphertexts(b *testing.B) {
	curve := elliptic.P256()
	rng := newDeterministicReader("bench-enc-sender")
	setup, err := GenerateCOSenderSetup(rng, curve)
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	wires := make([]Wire, 32)
	wireRand := newDeterministicReader("bench-enc-wires")
	for i := range wires {
		l0, err := readLabelFromReader(wireRand)
		if err != nil {
			b.Fatalf("label l0: %v", err)
		}
		l1, err := readLabelFromReader(wireRand)
		if err != nil {
			b.Fatalf("label l1: %v", err)
		}
		wires[i].L0 = l0
		wires[i].L1 = l1
	}
	bits := make([]bool, len(wires))
	for i := range bits {
		bits[i] = i%2 == 0
	}
	choiceRand := newDeterministicReader("bench-enc-choices")
	_, points, err := BuildCOChoices(choiceRand, curve, setup.Ax, setup.Ay, bits)
	if err != nil {
		b.Fatalf("BuildCOChoices: %v", err)
	}

	for b.Loop() {
		if _, err := EncryptCOCiphertexts(curve, setup, points, wires); err != nil {
			b.Fatalf("EncryptCOCiphertexts: %v", err)
		}
	}
}

// BenchmarkBuildCOChoices measures receiver choice preparation.
func BenchmarkBuildCOChoices(b *testing.B) {
	curve := elliptic.P256()
	setup, err := GenerateCOSenderSetup(newDeterministicReader("bench-build-sender"), curve)
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	bits := make([]bool, 32)
	for i := range bits {
		bits[i] = i%2 == 1
	}
	rng := newDeterministicReader("bench-build")
	for b.Loop() {
		if _, _, err := BuildCOChoices(rng, curve, setup.Ax, setup.Ay, bits); err != nil {
			b.Fatalf("BuildCOChoices: %v", err)
		}
	}
}

// BenchmarkDecryptCOCiphertexts measures helper decryption throughput.
func BenchmarkDecryptCOCiphertexts(b *testing.B) {
	curve := elliptic.P256()
	setup, err := GenerateCOSenderSetup(newDeterministicReader("bench-dec-sender"), curve)
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	wires := make([]Wire, 32)
	wireRand := newDeterministicReader("bench-dec-wires")
	for i := range wires {
		l0, err := readLabelFromReader(wireRand)
		if err != nil {
			b.Fatalf("label l0: %v", err)
		}
		l1, err := readLabelFromReader(wireRand)
		if err != nil {
			b.Fatalf("label l1: %v", err)
		}
		wires[i].L0 = l0
		wires[i].L1 = l1
	}
	bits := make([]bool, len(wires))
	for i := range bits {
		bits[i] = i%2 == 0
	}
	choiceRand := newDeterministicReader("bench-dec-choices")
	bundle, points, err := BuildCOChoices(choiceRand, curve, setup.Ax, setup.Ay, bits)
	if err != nil {
		b.Fatalf("BuildCOChoices: %v", err)
	}
	ciphertexts, err := EncryptCOCiphertexts(curve, setup, points, wires)
	if err != nil {
		b.Fatalf("EncryptCOCiphertexts: %v", err)
	}

	for b.Loop() {
		if _, err := DecryptCOCiphertexts(curve, bundle, ciphertexts); err != nil {
			b.Fatalf("DecryptCOCiphertexts: %v", err)
		}
	}
}

var benchmarkDeriveMaskSink [sha256.Size]byte

// BenchmarkDeriveMask measures the SHA-256 based mask derivation cost.
func BenchmarkDeriveMask(b *testing.B) {
	curve := elliptic.P256()
	scalar := new(big.Int).SetInt64(42)
	x, y := curve.ScalarBaseMult(scalar.Bytes())

	var id uint64
	for b.Loop() {
		benchmarkDeriveMaskSink = deriveMask(x, y, id)
		id++
	}
}
