package ot

import (
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/big"
)

// ErrNilCurve signals that a helper received a nil elliptic curve.
var ErrNilCurve = errors.New("ot: nil curve")

// ErrPointNotOnCurve signals that an input point is not on the active curve.
var ErrPointNotOnCurve = errors.New("ot: point not on curve")

// ECPoint describes an affine curve point.
type ECPoint struct {
	// X is the affine x-coordinate.
	X *big.Int

	// Y is the affine y-coordinate.
	Y *big.Int
}

// LabelCiphertext stores both encrypted labels for a single wire.
type LabelCiphertext struct {
	// Zero holds the ciphertext for the zero label.
	Zero LabelData

	// One holds the ciphertext for the one label.
	One LabelData
}

// COSenderSetup contains the immutable metadata sampled by the sender.
type COSenderSetup struct {
	// CurveName identifies the elliptic curve in use.
	CurveName string

	// Scalar stores the secret exponent 'a'.
	Scalar *big.Int

	// Ax is the affine x-coordinate for A = g^a.
	Ax *big.Int

	// Ay is the affine y-coordinate for A = g^a.
	Ay *big.Int

	// AaInvX stores the affine x-coordinate of A^{-a}.
	AaInvX *big.Int

	// AaInvY stores the affine y-coordinate of A^{-a}.
	AaInvY *big.Int
}

// COChoiceBundle preserves the receiver-side secrets for later decryption.
type COChoiceBundle struct {
	// CurveName identifies the elliptic curve in use.
	CurveName string

	// Ax stores the sender's affine x-coordinate.
	Ax *big.Int

	// Ay stores the sender's affine y-coordinate.
	Ay *big.Int

	// Scalars contains the receiver's random scalars.
	Scalars []*big.Int

	// Bits mirrors the receiver's choice bits.
	Bits []bool
}

// GenerateCOSenderSetup samples the sender randomness and curve points.
func GenerateCOSenderSetup(rand io.Reader, curve elliptic.Curve) (COSenderSetup, error) {
	if curve == nil {
		return COSenderSetup{}, ErrNilCurve
	}
	params := curve.Params()

	a, err := crand.Int(rand, params.N)
	if err != nil {
		return COSenderSetup{}, err
	}
	Ax, Ay := curve.ScalarBaseMult(a.Bytes())
	Aax, Aay := curve.ScalarMult(Ax, Ay, a.Bytes())

	AaInvx := big.NewInt(0).Set(Aax)
	AaInvy := big.NewInt(0).Sub(params.P, Aay)

	return COSenderSetup{
		CurveName: curve.Params().Name,
		Scalar:    a,
		Ax:        Ax,
		Ay:        Ay,
		AaInvX:    AaInvx,
		AaInvY:    AaInvy,
	}, nil
}

// EncryptCOCiphertexts encrypts wire labels for every evaluator input bit.
func EncryptCOCiphertexts(curve elliptic.Curve, setup COSenderSetup, points []ECPoint, wires []Wire) ([]LabelCiphertext, error) {
	if curve == nil {
		return nil, ErrNilCurve
	}
	if err := ensureOnCurve(curve, setup.Ax, setup.Ay); err != nil {
		return nil, err
	}
	if len(points) != len(wires) {
		return nil, fmt.Errorf("OT point count mismatch: got %d want %d", len(points), len(wires))
	}

	aBytes := setup.Scalar.Bytes()

	result := make([]LabelCiphertext, len(points))
	for idx, point := range points {
		if err := ensureOnCurve(curve, point.X, point.Y); err != nil {
			return nil, err
		}
		Bx, By := curve.ScalarMult(point.X, point.Y, aBytes)
		Bax, Bay := curve.Add(Bx, By, setup.AaInvX, setup.AaInvY)

		mask0 := deriveMask(Bx, By, uint64(idx))
		mask1 := deriveMask(Bax, Bay, uint64(idx))

		var tmp LabelData
		wires[idx].L0.GetData(&tmp)
		copy(result[idx].Zero[:], xor(mask0[:], tmp[:]))

		wires[idx].L1.GetData(&tmp)
		copy(result[idx].One[:], xor(mask1[:], tmp[:]))
	}

	return result, nil
}

// BuildCOChoices constructs the receiver EC points for each choice bit.
func BuildCOChoices(rand io.Reader, curve elliptic.Curve, Ax, Ay *big.Int, bits []bool) (COChoiceBundle, []ECPoint, error) {
	if curve == nil {
		return COChoiceBundle{}, nil, ErrNilCurve
	}
	if err := ensureOnCurve(curve, Ax, Ay); err != nil {
		return COChoiceBundle{}, nil, err
	}
	params := curve.Params()
	points := make([]ECPoint, len(bits))
	scalars := make([]*big.Int, len(bits))
	for idx, bit := range bits {
		b, err := crand.Int(rand, params.N)
		if err != nil {
			return COChoiceBundle{}, nil, err
		}
		scalars[idx] = b

		Bx, By := curve.ScalarBaseMult(b.Bytes())
		if bit {
			Bx, By = curve.Add(Bx, By, Ax, Ay)
		}

		points[idx] = ECPoint{
			X: Bx,
			Y: By,
		}
	}

	bundle := COChoiceBundle{
		CurveName: curve.Params().Name,
		Ax:        new(big.Int).Set(Ax),
		Ay:        new(big.Int).Set(Ay),
		Scalars:   scalars,
		Bits:      append([]bool(nil), bits...),
	}

	return bundle, points, nil
}

// ensureOnCurve verifies that (x,y) is a valid affine point on the curve.
func ensureOnCurve(curve elliptic.Curve, x, y *big.Int) error {
	if curve == nil {
		return ErrNilCurve
	}
	if x == nil || y == nil || !curve.IsOnCurve(x, y) {
		return ErrPointNotOnCurve
	}
	return nil
}

// DecryptCOCiphertexts decodes the chosen labels from ciphertexts.
func DecryptCOCiphertexts(curve elliptic.Curve, bundle COChoiceBundle, data []LabelCiphertext) ([]Label, error) {
	if curve == nil {
		return nil, ErrNilCurve
	}

	count := len(bundle.Bits)
	if len(bundle.Scalars) != count || len(data) != count {
		return nil, fmt.Errorf("invalid CO ciphertext bundle")
	}

	result := make([]Label, count)
	for idx := 0; idx < count; idx++ {
		Asx, Asy := curve.ScalarMult(bundle.Ax, bundle.Ay, bundle.Scalars[idx].Bytes())
		mask := deriveMask(Asx, Asy, uint64(idx))

		var cipher []byte
		if bundle.Bits[idx] {
			cipher = data[idx].One[:]
		} else {
			cipher = data[idx].Zero[:]
		}

		var tmp LabelData
		copy(tmp[:], xor(mask[:], cipher))
		result[idx].SetData(&tmp)
	}

	return result, nil
}

// deriveMask derives the XOR pad for a particular Diffie-Hellman output.
func deriveMask(x, y *big.Int, id uint64) [sha256.Size]byte {
	hash := sha256.New()
	hash.Write(x.Bytes())
	hash.Write(y.Bytes())

	var idBuf [8]byte
	bo.PutUint64(idBuf[:], id)
	hash.Write(idBuf[:])

	var sum [sha256.Size]byte
	hash.Sum(sum[:0])

	return sum
}

// xor returns dst XOR src (truncating to the shorter slice).
func xor(dst, src []byte) []byte {
	l := len(dst)
	if len(src) < l {
		l = len(src)
	}

	for i := 0; i < l; i++ {
		dst[i] ^= src[i]
	}

	return dst[:l]
}
