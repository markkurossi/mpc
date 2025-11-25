package sha2pc

import (
	"bytes"
	"crypto/elliptic"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/markkurossi/mpc/ot"
)

const (
	// magicRound1 tags Round1 payload encodings.
	magicRound1 = "R1"

	// magicRound2 tags Round2 payload encodings.
	magicRound2 = "R2"

	// magicRound3 tags Round3 payload encodings.
	magicRound3 = "R3"

	// magicGarblerSession tags garbler session encodings.
	magicGarblerSession = "GS"

	// magicEvalSession tags evaluator session encodings.
	magicEvalSession = "ES"
)

// chunkSizeLimit bounds a single encoded chunk used for variable-length metadata.
const chunkSizeLimit = 1 * 1024 * 1024

// EncodeRound1 turns a Round1Payload into bytes.
func EncodeRound1(curve elliptic.Curve, p Round1Payload) ([]byte, error) {
	if curve == nil {
		return nil, errNilCurve
	}
	var buf bytes.Buffer
	buf.Write([]byte(magicRound1))
	var sid [8]byte
	binary.BigEndian.PutUint64(sid[:], p.SessionID)
	buf.Write(sid[:])
	if err := encodeOTSetup(&buf, curve, p.OT); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecodeRound1 reconstructs a Round1Payload from bytes.
func DecodeRound1(curve elliptic.Curve, data []byte) (Round1Payload, error) {
	if curve == nil {
		return Round1Payload{}, errNilCurve
	}
	reader := bytes.NewReader(data)
	var payload Round1Payload
	magic := make([]byte, 2)
	if _, err := io.ReadFull(reader, magic); err != nil {
		return Round1Payload{}, err
	}
	if string(magic) != magicRound1 {
		return Round1Payload{}, fmt.Errorf("invalid round1 magic")
	}
	var sidBuf [8]byte
	if _, err := io.ReadFull(reader, sidBuf[:]); err != nil {
		return Round1Payload{}, err
	}
	var err error
	payload.OT, err = decodeOTSetup(curve, reader)
	if err != nil {
		return Round1Payload{}, err
	}
	if payload.OT.CurveName != curve.Params().Name {
		return Round1Payload{}, fmt.Errorf("sha2pc: round1 curve mismatch %s vs %s",
			payload.OT.CurveName, curve.Params().Name)
	}
	payload.SessionID = binary.BigEndian.Uint64(sidBuf[:])

	return payload, nil
}

// EncodeRound2 turns a Round2Payload into bytes.
func EncodeRound2(curve elliptic.Curve, p Round2Payload) ([]byte, error) {
	if curve == nil {
		return nil, errNilCurve
	}
	name := curve.Params().Name
	var buf bytes.Buffer
	buf.Write([]byte(magicRound2))
	var sid [8]byte
	binary.BigEndian.PutUint64(sid[:], p.SessionID)
	buf.Write(sid[:])
	writeChunk(&buf, []byte(name))
	if err := encodePoints(curve, &buf, p.Choices); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecodeRound2 reconstructs a Round2Payload.
func DecodeRound2(curve elliptic.Curve, data []byte) (Round2Payload, error) {
	if curve == nil {
		return Round2Payload{}, errNilCurve
	}
	reader := bytes.NewReader(data)
	magic := make([]byte, 2)
	if _, err := io.ReadFull(reader, magic); err != nil {
		return Round2Payload{}, err
	}
	if string(magic) != magicRound2 {
		return Round2Payload{}, fmt.Errorf("invalid round2 magic")
	}

	var sidBuf [8]byte
	if _, err := io.ReadFull(reader, sidBuf[:]); err != nil {
		return Round2Payload{}, err
	}

	nameChunk, err := readChunk(reader)
	if err != nil {
		return Round2Payload{}, err
	}
	curveName := string(nameChunk)
	if curveName != curve.Params().Name {
		return Round2Payload{}, fmt.Errorf("sha2pc: round2 curve mismatch %s vs %s",
			curveName, curve.Params().Name)
	}

	rest, err := io.ReadAll(reader)
	if err != nil {
		return Round2Payload{}, err
	}

	points, err := decodePoints(curve, rest)
	if err != nil {
		return Round2Payload{}, err
	}

	return Round2Payload{
		SessionID: binary.BigEndian.Uint64(sidBuf[:]),
		CurveName: curveName,
		Choices:   points,
	}, nil
}

// EncodeRound3 turns a Round3Payload into bytes.
func EncodeRound3(p Round3Payload) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write([]byte(magicRound3))
	var sid [8]byte
	binary.BigEndian.PutUint64(sid[:], p.SessionID)
	buf.Write(sid[:])
	buf.Write(p.Key[:])
	if err := encodeGarbledTables(&buf, p.GarbledTables); err != nil {
		return nil, err
	}
	if err := encodeLabels(&buf, p.GarblerInputs); err != nil {
		return nil, err
	}
	if err := encodeOutputHints(&buf, p.OutputHints); err != nil {
		return nil, err
	}
	if err := encodeCiphertexts(&buf, p.Ciphertexts); err != nil {
		return nil, err
	}
	if buf.Len() != round3PayloadLen {
		return nil, fmt.Errorf("sha2pc: round3 length mismatch: produced %d bytes want %d",
			buf.Len(), round3PayloadLen)
	}

	return buf.Bytes(), nil
}

// DecodeRound3 reconstructs a Round3Payload from bytes.
func DecodeRound3(data []byte) (Round3Payload, error) {
	if len(data) != round3PayloadLen {
		return Round3Payload{}, fmt.Errorf("sha2pc: round3 payload mismatch: got %d want %d",
			len(data), round3PayloadLen)
	}
	var payload Round3Payload
	offset := 0
	if string(data[offset:offset+len(magicRound3)]) != magicRound3 {
		return Round3Payload{}, fmt.Errorf("invalid round3 magic")
	}
	offset += len(magicRound3)
	payload.SessionID = binary.BigEndian.Uint64(data[offset : offset+sessionIDBytes])
	offset += sessionIDBytes
	copy(payload.Key[:], data[offset:offset+garblingKeyBytes])
	offset += garblingKeyBytes

	end := offset + garbledTableByteLen
	var err error
	payload.GarbledTables, err = decodeGarbledTables(data[offset:end])
	if err != nil {
		return Round3Payload{}, err
	}
	offset = end

	end = offset + garblerInputLabelBytes
	payload.GarblerInputs, err = decodeLabels(data[offset:end])
	if err != nil {
		return Round3Payload{}, err
	}
	offset = end

	end = offset + outputHintBytes
	payload.OutputHints, err = decodeOutputHints(data[offset:end])
	if err != nil {
		return Round3Payload{}, err
	}
	offset = end

	end = offset + ciphertextBytes
	payload.Ciphertexts, err = decodeCiphertexts(data[offset:end])
	if err != nil {
		return Round3Payload{}, err
	}

	return payload, nil
}

// EncodeGarblerSession serializes a GarblerSession for persistence.
func EncodeGarblerSession(curve elliptic.Curve, session *GarblerSession) ([]byte, error) {
	if session == nil {
		return nil, fmt.Errorf("nil garbler session")
	}

	var buf bytes.Buffer
	buf.Write([]byte(magicGarblerSession))
	var sid [8]byte
	binary.BigEndian.PutUint64(sid[:], session.SessionID)
	buf.Write(sid[:])
	setup, err := encodeCOSenderSetup(curve, session.SenderSetup)
	if err != nil {
		return nil, err
	}
	writeChunk(&buf, setup)

	return buf.Bytes(), nil
}

// DecodeGarblerSession reconstructs a GarblerSession from bytes.
func DecodeGarblerSession(curve elliptic.Curve, data []byte) (*GarblerSession, error) {
	if curve == nil {
		return nil, errNilCurve
	}
	reader := bytes.NewReader(data)
	var session GarblerSession
	magic := make([]byte, 2)
	if _, err := io.ReadFull(reader, magic); err != nil {
		return nil, err
	}
	if string(magic) != magicGarblerSession {
		return nil, fmt.Errorf("invalid garbler session magic")
	}
	var sidBuf [8]byte
	if _, err := io.ReadFull(reader, sidBuf[:]); err != nil {
		return nil, err
	}

	chunk, err := readChunk(reader)
	if err != nil {
		return nil, err
	}
	session.SenderSetup, err = decodeCOSenderSetup(curve, chunk)
	if err != nil {
		return nil, err
	}

	session.SessionID = binary.BigEndian.Uint64(sidBuf[:])

	return &session, nil
}

// EncodeEvaluatorSession serializes an EvaluatorSession for persistence.
func EncodeEvaluatorSession(curve elliptic.Curve, session *EvaluatorSession) ([]byte, error) {
	if session == nil {
		return nil, fmt.Errorf("nil evaluator session")
	}

	var buf bytes.Buffer
	buf.Write([]byte(magicEvalSession))
	var sid [8]byte
	binary.BigEndian.PutUint64(sid[:], session.SessionID)
	buf.Write(sid[:])
	data, err := encodeChoiceBundle(curve, session.ChoiceBundle)
	if err != nil {
		return nil, err
	}
	writeChunk(&buf, data)

	return buf.Bytes(), nil
}

// DecodeEvaluatorSession reconstructs an EvaluatorSession from bytes.
func DecodeEvaluatorSession(curve elliptic.Curve, data []byte) (*EvaluatorSession, error) {
	if curve == nil {
		return nil, errNilCurve
	}
	reader := bytes.NewReader(data)
	var session EvaluatorSession
	magic := make([]byte, 2)
	if _, err := io.ReadFull(reader, magic); err != nil {
		return nil, err
	}
	if string(magic) != magicEvalSession {
		return nil, fmt.Errorf("invalid evaluator session magic")
	}
	var sidBuf [8]byte
	if _, err := io.ReadFull(reader, sidBuf[:]); err != nil {
		return nil, err
	}

	chunk, err := readChunk(reader)
	if err != nil {
		return nil, err
	}
	session.ChoiceBundle, err = decodeChoiceBundle(curve, chunk)
	if err != nil {
		return nil, err
	}

	session.SessionID = binary.BigEndian.Uint64(sidBuf[:])

	return &session, nil
}

// encodeLabels flattens all labels into raw bytes.
func encodeLabels(buf *bytes.Buffer, labels []ot.Label) error {
	if len(labels) != garblerInputLabelCount {
		return fmt.Errorf("sha2pc: garbler label mismatch %d != %d",
			len(labels), garblerInputLabelCount)
	}
	var tmp ot.LabelData
	for _, label := range labels {
		label.GetData(&tmp)
		buf.Write(tmp[:])
	}
	return nil
}

// decodeLabels rebuilds labels from their byte form.
func decodeLabels(data []byte) ([]ot.Label, error) {
	if len(data) != garblerInputLabelBytes {
		return nil, fmt.Errorf("label buffer mismatch: %d != %d",
			len(data), garblerInputLabelBytes)
	}
	const labelSize = labelByteLen
	var tmp ot.LabelData
	count := len(data) / labelSize
	result := make([]ot.Label, count)
	for i := 0; i < count; i++ {
		copy(tmp[:], data[i*labelSize:(i+1)*labelSize])
		result[i].SetData(&tmp)
	}

	return result, nil
}

// encodeGarbledTables serializes garbled table rows directly into buf.
func encodeGarbledTables(buf *bytes.Buffer, tables [][]ot.Label) error {
	if len(tables) != len(sha256xorCircuit.Gates) {
		return fmt.Errorf("sha2pc: garbled table count mismatch %d vs %d",
			len(tables), len(sha256xorCircuit.Gates))
	}
	var tmp ot.LabelData
	written := 0
	for idx, gate := range sha256xorCircuit.Gates {
		want, err := gateCiphertextCount(gate.Op)
		if err != nil {
			return err
		}
		row := tables[idx]
		if want == 0 {
			if len(row) != 0 {
				return fmt.Errorf("sha2pc: gate %d expected 0 labels got %d",
					idx, len(row))
			}
			continue
		}
		if len(row) != want {
			return fmt.Errorf("sha2pc: gate %d expected %d labels got %d",
				idx, want, len(row))
		}
	}
	for idx, gate := range sha256xorCircuit.Gates {
		want, err := gateCiphertextCount(gate.Op)
		if err != nil {
			return err
		}
		row := tables[idx]
		if want == 0 {
			continue
		}
		for _, label := range row {
			label.GetData(&tmp)
			buf.Write(tmp[:])
			written += len(tmp)
		}
	}
	if written != garbledTableByteLen {
		return fmt.Errorf("sha2pc: wrote %d garbled-table bytes, expected %d",
			written, garbledTableByteLen)
	}

	return nil
}

// decodeGarbledTables reconstructs table rows from bytes.
func decodeGarbledTables(data []byte) ([][]ot.Label, error) {
	if len(data) != garbledTableByteLen {
		return nil, fmt.Errorf("sha2pc: garbled table buffer mismatch: got %d want %d",
			len(data), garbledTableByteLen)
	}
	reader := bytes.NewReader(data)
	result := make([][]ot.Label, len(sha256xorCircuit.Gates))
	var tmp ot.LabelData
	for idx, gate := range sha256xorCircuit.Gates {
		want, err := gateCiphertextCount(gate.Op)
		if err != nil {
			return nil, err
		}
		if want == 0 {
			continue
		}
		row := make([]ot.Label, want)
		for j := 0; j < want; j++ {
			if _, err := reader.Read(tmp[:]); err != nil {
				return nil, err
			}
			row[j].SetData(&tmp)
		}
		result[idx] = row
	}
	return result, nil
}

// encodeOutputHints serializes every output wire.
func encodeOutputHints(buf *bytes.Buffer, wires []ot.Wire) error {
	if len(wires) != outputHintCount {
		return fmt.Errorf("sha2pc: output hint mismatch %d != %d",
			len(wires), outputHintCount)
	}
	var tmp ot.LabelData
	for _, wire := range wires {
		wire.L0.GetData(&tmp)
		buf.Write(tmp[:])
		wire.L1.GetData(&tmp)
		buf.Write(tmp[:])
	}
	return nil
}

// decodeOutputHints rebuilds output wires.
func decodeOutputHints(data []byte) ([]ot.Wire, error) {
	if len(data) != outputHintBytes {
		return nil, fmt.Errorf("output hint buffer mismatch: %d != %d",
			len(data), outputHintBytes)
	}
	var tmp ot.LabelData
	result := make([]ot.Wire, outputHintCount)
	reader := bytes.NewReader(data)
	for i := 0; i < outputHintCount; i++ {
		if _, err := reader.Read(tmp[:]); err != nil {
			return nil, err
		}
		result[i].L0.SetData(&tmp)
		if _, err := reader.Read(tmp[:]); err != nil {
			return nil, err
		}
		result[i].L1.SetData(&tmp)
	}

	return result, nil
}

// encodePoints serializes EC points into buf using fixed-width X coordinates
// plus packed Y parities to keep sizes uniform regardless of random inputs.
func encodePoints(curve elliptic.Curve, buf *bytes.Buffer, points []ot.ECPoint) error {
	if len(points) != evaluatorCiphertextCount {
		return fmt.Errorf("sha2pc: choice count mismatch %d != %d",
			len(points), evaluatorCiphertextCount)
	}
	byteLen, err := curveByteLen(curve)
	if err != nil {
		return err
	}
	for _, p := range points {
		writeFixedBigInt(buf, byteLen, p.X)
	}

	signs := packPointSigns(points)
	if len(signs) != evaluatorChoiceSignBytes {
		return fmt.Errorf("sha2pc: sign buffer mismatch %d != %d",
			len(signs), evaluatorChoiceSignBytes)
	}
	buf.Write(signs)

	return nil
}

// decodePoints rebuilds EC points from compressed X coordinates and sign bits.
func decodePoints(curve elliptic.Curve, data []byte) ([]ot.ECPoint, error) {
	if curve == nil {
		return nil, errNilCurve
	}
	byteLen, err := curveByteLen(curve)
	if err != nil {
		return nil, err
	}
	expected := evaluatorCiphertextCount*byteLen + evaluatorChoiceSignBytes
	if len(data) != expected {
		return nil, fmt.Errorf("round2 choice payload mismatch: got %d want %d", len(data), expected)
	}
	xs := make([]*big.Int, evaluatorCiphertextCount)
	offset := 0
	for i := 0; i < evaluatorCiphertextCount; i++ {
		xs[i] = new(big.Int).SetBytes(data[offset : offset+byteLen])
		offset += byteLen
	}
	signs := data[offset:]
	result := make([]ot.ECPoint, evaluatorCiphertextCount)
	for i := 0; i < evaluatorCiphertextCount; i++ {
		odd := pointSign(signs, i)
		compressed := make([]byte, 1+byteLen)
		if odd {
			compressed[0] = 0x03
		} else {
			compressed[0] = 0x02
		}
		xBytes := xs[i].Bytes()
		copy(compressed[1+byteLen-len(xBytes):], xBytes)
		x, y := elliptic.UnmarshalCompressed(curve, compressed)
		if x == nil || y == nil {
			return nil, fmt.Errorf("failed to decompress evaluator choice %d", i)
		}
		result[i] = ot.ECPoint{X: x, Y: y}
	}

	return result, nil
}

// packPointSigns packs the parity of the Y coordinate for each point.
func packPointSigns(points []ot.ECPoint) []byte {
	if len(points) == 0 {
		return nil
	}
	signs := make([]byte, (len(points)+7)/8)
	for i, p := range points {
		if p.Y.Bit(0) == 1 {
			signs[i/8] |= 1 << uint(i%8)
		}
	}

	return signs
}

// pointSign fetches the stored parity bit for the ith point.
func pointSign(signs []byte, idx int) bool {
	if len(signs) == 0 {
		return false
	}

	b := signs[idx/8]
	return (b & (1 << uint(idx%8))) != 0
}

// encodeCiphertexts serializes OT ciphertexts.
func encodeCiphertexts(buf *bytes.Buffer, ct []ot.LabelCiphertext) error {
	if len(ct) != evaluatorCiphertextCount {
		return fmt.Errorf("sha2pc: ciphertext count mismatch %d != %d",
			len(ct), evaluatorCiphertextCount)
	}
	for _, c := range ct {
		buf.Write(c.Zero[:])
		buf.Write(c.One[:])
	}
	return nil
}

// decodeCiphertexts rebuilds OT ciphertexts.
func decodeCiphertexts(data []byte) ([]ot.LabelCiphertext, error) {
	if len(data) != ciphertextBytes {
		return nil, fmt.Errorf("ciphertext buffer mismatch: %d != %d",
			len(data), ciphertextBytes)
	}
	reader := bytes.NewReader(data)
	result := make([]ot.LabelCiphertext, evaluatorCiphertextCount)
	for i := 0; i < evaluatorCiphertextCount; i++ {
		if _, err := reader.Read(result[i].Zero[:]); err != nil {
			return nil, err
		}
		if _, err := reader.Read(result[i].One[:]); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// encodeOTSetup serializes the OT sender setup.
func encodeOTSetup(buf *bytes.Buffer, curve elliptic.Curve, otSetup OTSenderSetup) error {
	if curve == nil {
		return errNilCurve
	}
	name := curve.Params().Name
	if otSetup.CurveName == "" {
		otSetup.CurveName = name
	}
	if otSetup.CurveName != name {
		return fmt.Errorf("sha2pc: OT setup curve mismatch %s vs %s",
			otSetup.CurveName, name)
	}
	byteLen, err := curveByteLen(curve)
	if err != nil {
		return err
	}
	writeChunk(buf, []byte(otSetup.CurveName))
	writeFixedBigInt(buf, byteLen, otSetup.A.X)
	writeFixedBigInt(buf, byteLen, otSetup.A.Y)

	return nil
}

// decodeOTSetup rebuilds the OT sender setup.
func decodeOTSetup(curve elliptic.Curve, reader *bytes.Reader) (OTSenderSetup, error) {
	if curve == nil {
		return OTSenderSetup{}, errNilCurve
	}
	name, err := readChunk(reader)
	if err != nil {
		return OTSenderSetup{}, err
	}
	if string(name) != curve.Params().Name {
		return OTSenderSetup{}, fmt.Errorf("sha2pc: OT setup curve mismatch %s vs %s",
			string(name), curve.Params().Name)
	}
	byteLen, err := curveByteLen(curve)
	if err != nil {
		return OTSenderSetup{}, err
	}
	x, err := readFixedBigInt(reader, byteLen)
	if err != nil {
		return OTSenderSetup{}, err
	}
	y, err := readFixedBigInt(reader, byteLen)
	if err != nil {
		return OTSenderSetup{}, err
	}

	return OTSenderSetup{
		CurveName: string(name),
		A: ot.ECPoint{
			X: x,
			Y: y,
		},
	}, nil
}

// writeChunk writes a length-prefixed byte slice.
func writeChunk(buf *bytes.Buffer, data []byte) {
	var scratch [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(scratch[:], uint64(len(data)))
	buf.Write(scratch[:n])
	buf.Write(data)
}

// readChunk reads a single length-prefixed byte slice.
func readChunk(r *bytes.Reader) ([]byte, error) {
	length, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	if length > chunkSizeLimit {
		return nil, fmt.Errorf("chunk length %d exceeds limit %d", length, chunkSizeLimit)
	}
	if int64(length) > int64(r.Len()) {
		return nil, fmt.Errorf("chunk length %d exceeds remaining %d", length, r.Len())
	}
	data := make([]byte, length)
	if _, err := r.Read(data); err != nil {
		return nil, err
	}

	return data, nil
}

// writeBigInt writes a length-prefixed big integer.
func writeBigInt(buf *bytes.Buffer, v *big.Int) {
	if v == nil {
		writeChunk(buf, nil)
		return
	}
	writeChunk(buf, v.Bytes())
}

// readBigInt reads a length-prefixed big integer.
func readBigInt(r *bytes.Reader) (*big.Int, error) {
	data, err := readChunk(r)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return big.NewInt(0), nil
	}

	return new(big.Int).SetBytes(data), nil
}

// writeFixedBigInt writes v into buf using exactly byteLen big-endian bytes.
func writeFixedBigInt(buf *bytes.Buffer, byteLen int, v *big.Int) {
	tmp := make([]byte, byteLen)
	if v != nil {
		value := v.Bytes()
		copy(tmp[byteLen-len(value):], value)
	}
	buf.Write(tmp)
}

// readFixedBigInt reads a fixed-width big integer without length prefix.
func readFixedBigInt(r *bytes.Reader, byteLen int) (*big.Int, error) {
	tmp := make([]byte, byteLen)
	if _, err := io.ReadFull(r, tmp); err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(tmp), nil
}

// curveByteLen returns the size in bytes of the provided curve's base field.
func curveByteLen(curve elliptic.Curve) (int, error) {
	if curve == nil {
		return 0, errNilCurve
	}

	return (curve.Params().BitSize + 7) / 8, nil
}

// encodeCOSenderSetup serializes the CO sender setup.
func encodeCOSenderSetup(curve elliptic.Curve, setup ot.COSenderSetup) ([]byte, error) {
	if curve == nil {
		return nil, errNilCurve
	}
	name := curve.Params().Name
	if setup.CurveName == "" {
		setup.CurveName = name
	}
	if setup.CurveName != name {
		return nil, fmt.Errorf("sha2pc: CO sender curve mismatch %s vs %s",
			setup.CurveName, name)
	}
	byteLen, err := curveByteLen(curve)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writeChunk(&buf, []byte(setup.CurveName))
	writeFixedBigInt(&buf, byteLen, setup.Scalar)
	writeFixedBigInt(&buf, byteLen, setup.Ax)
	writeFixedBigInt(&buf, byteLen, setup.Ay)
	writeFixedBigInt(&buf, byteLen, setup.AaInvX)
	writeFixedBigInt(&buf, byteLen, setup.AaInvY)

	return buf.Bytes(), nil
}

// decodeCOSenderSetup rebuilds a CO sender setup from bytes.
func decodeCOSenderSetup(curve elliptic.Curve, data []byte) (ot.COSenderSetup, error) {
	if curve == nil {
		return ot.COSenderSetup{}, errNilCurve
	}
	reader := bytes.NewReader(data)

	name, err := readChunk(reader)
	if err != nil {
		return ot.COSenderSetup{}, err
	}
	if string(name) != curve.Params().Name {
		return ot.COSenderSetup{}, fmt.Errorf("sha2pc: CO sender curve mismatch %s vs %s",
			string(name), curve.Params().Name)
	}

	byteLen, err := curveByteLen(curve)
	if err != nil {
		return ot.COSenderSetup{}, err
	}

	scalar, err := readFixedBigInt(reader, byteLen)
	if err != nil {
		return ot.COSenderSetup{}, err
	}

	ax, err := readFixedBigInt(reader, byteLen)
	if err != nil {
		return ot.COSenderSetup{}, err
	}

	ay, err := readFixedBigInt(reader, byteLen)
	if err != nil {
		return ot.COSenderSetup{}, err
	}

	ainvx, err := readFixedBigInt(reader, byteLen)
	if err != nil {
		return ot.COSenderSetup{}, err
	}

	ainvy, err := readFixedBigInt(reader, byteLen)
	if err != nil {
		return ot.COSenderSetup{}, err
	}

	return ot.COSenderSetup{
		CurveName: string(name),
		Scalar:    scalar,
		Ax:        ax,
		Ay:        ay,
		AaInvX:    ainvx,
		AaInvY:    ainvy,
	}, nil
}

// encodeChoiceBundle serializes a CO choice bundle.
func encodeChoiceBundle(curve elliptic.Curve, bundle ot.COChoiceBundle) ([]byte, error) {
	if curve == nil {
		return nil, errNilCurve
	}
	name := curve.Params().Name
	if bundle.CurveName == "" {
		bundle.CurveName = name
	}
	if bundle.CurveName != name {
		return nil, fmt.Errorf("sha2pc: CO choice curve mismatch %s vs %s",
			bundle.CurveName, name)
	}
	byteLen, err := curveByteLen(curve)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writeChunk(&buf, []byte(bundle.CurveName))
	writeFixedBigInt(&buf, byteLen, bundle.Ax)
	writeFixedBigInt(&buf, byteLen, bundle.Ay)

	if len(bundle.Scalars) != evaluatorCiphertextCount {
		return nil, fmt.Errorf("sha2pc: choice scalar mismatch %d != %d",
			len(bundle.Scalars), evaluatorCiphertextCount)
	}
	if len(bundle.Bits) != evaluatorCiphertextCount {
		return nil, fmt.Errorf("sha2pc: choice bits mismatch %d != %d",
			len(bundle.Bits), evaluatorCiphertextCount)
	}
	for _, scalar := range bundle.Scalars {
		writeFixedBigInt(&buf, byteLen, scalar)
	}

	bitBytes := bitsToBytesLittle(bundle.Bits)
	if len(bitBytes) != evaluatorChoiceSignBytes {
		return nil, fmt.Errorf("sha2pc: choice bit encoding mismatch %d != %d",
			len(bitBytes), evaluatorChoiceSignBytes)
	}
	buf.Write(bitBytes)

	return buf.Bytes(), nil
}

// decodeChoiceBundle restores a CO choice bundle from bytes.
func decodeChoiceBundle(curve elliptic.Curve, data []byte) (ot.COChoiceBundle, error) {
	if curve == nil {
		return ot.COChoiceBundle{}, errNilCurve
	}
	reader := bytes.NewReader(data)

	name, err := readChunk(reader)
	if err != nil {
		return ot.COChoiceBundle{}, err
	}
	if string(name) != curve.Params().Name {
		return ot.COChoiceBundle{}, fmt.Errorf("sha2pc: CO choice curve mismatch %s vs %s",
			string(name), curve.Params().Name)
	}
	byteLen, err := curveByteLen(curve)
	if err != nil {
		return ot.COChoiceBundle{}, err
	}

	ax, err := readFixedBigInt(reader, byteLen)
	if err != nil {
		return ot.COChoiceBundle{}, err
	}

	ay, err := readFixedBigInt(reader, byteLen)
	if err != nil {
		return ot.COChoiceBundle{}, err
	}

	scalars := make([]*big.Int, evaluatorCiphertextCount)
	for i := 0; i < evaluatorCiphertextCount; i++ {
		value, err := readFixedBigInt(reader, byteLen)
		if err != nil {
			return ot.COChoiceBundle{}, err
		}
		scalars[i] = value
	}

	raw := make([]byte, evaluatorChoiceSignBytes)
	if _, err := reader.Read(raw); err != nil {
		return ot.COChoiceBundle{}, err
	}
	bits := bytesToBitsLittle(raw)
	if len(bits) < evaluatorCiphertextCount {
		return ot.COChoiceBundle{}, fmt.Errorf("choice bit buffer too short: %d", len(bits))
	}
	bits = bits[:evaluatorCiphertextCount]

	return ot.COChoiceBundle{
		CurveName: string(name),
		Ax:        ax,
		Ay:        ay,
		Scalars:   scalars,
		Bits:      bits,
	}, nil
}
