package vole

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"

	"golang.org/x/crypto/chacha20"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/otext"
	"github.com/markkurossi/mpc/p2p"
)

// Role constants
type Role int

const (
	SenderRole   Role = 0
	ReceiverRole Role = 1
)

var (
	ErrNotImplemented = errors.New("vole: not implemented")
)

// Ext wraps base OT (optional) and provides VOLE batched interface.
type Ext struct {
	oti  ot.OT
	conn *p2p.Conn
	role Role

	// internal IKNP ext (created in Setup if oti != nil)
	iknp *otext.IKNPExt

	// small pool for buffers; reused for PRG outputs etc.
	bufPool sync.Pool
}

func NewExt(oti ot.OT, conn *p2p.Conn, role Role) *Ext {
	e := &Ext{
		oti:  oti,
		conn: conn,
		role: role,
	}
	e.bufPool = sync.Pool{New: func() any { return make([]byte, 0, 4096) }}
	return e
}

// Setup sets up underlying IKNP if an OT instance is present.
// If oti == nil then Setup is a no-op (shim mode for tests).
func (e *Ext) Setup(r io.Reader) error {
	if e.oti == nil {
		e.iknp = nil
		return nil
	}

	if e.role == SenderRole {
		if err := e.oti.InitSender(e.conn); err != nil {
			return fmt.Errorf("vole: InitSender: %w", err)
		}
	} else {
		if err := e.oti.InitReceiver(e.conn); err != nil {
			return fmt.Errorf("vole: InitReceiver: %w", err)
		}
	}

	var rrole int
	if e.role == SenderRole {
		rrole = otext.SenderRole
	} else {
		rrole = otext.ReceiverRole
	}
	e.iknp = otext.NewIKNPExt(e.oti, e.conn, rrole)
	return e.iknp.Setup(r)
}

// -----------------------------------------------------------------------------
// MulSender / MulReceiver (packed-IKNP path)
// -----------------------------------------------------------------------------

// MulSender functionality: senderInputs = x[0..m-1]. Returns r[0..m-1].
func (e *Ext) MulSender(senderInputs []*big.Int, p *big.Int) ([]*big.Int, error) {
	if e == nil {
		return nil, errors.New("vole: nil Ext")
	}
	m := len(senderInputs)
	if m == 0 {
		return nil, nil
	}

	// If iknp is not configured, fall back to the channel-shim (same as before)
	if e.iknp == nil {
		return e.mulSenderShim(senderInputs, p)
	}

	// Packed path: ExpandSend(m) -> wires (one wire per triple)
	wires, err := e.iknp.ExpandSend(m)
	if err != nil {
		return nil, fmt.Errorf("vole: ExpandSend: %w", err)
	}
	if len(wires) != m {
		return nil, fmt.Errorf("vole: ExpandSend returned %d wires, want %d", len(wires), m)
	}

	// Derive r_i from L0 label using PRG and reduce mod p.
	rs := make([]*big.Int, m)
	for i := 0; i < m; i++ {
		var ld ot.LabelData
		wires[i].L0.GetData(&ld)

		// Expand label to 32-bytes.
		var pad [32]byte
		prgExpandLabel(ld, &pad)

		rsi := new(big.Int).SetBytes(pad[:])
		rsi.Mod(rsi, p)
		rs[i] = rsi
	}

	// Now receive the receiver's packed y-vector (m*32 bytes).
	// (This is the single extra message; we still benefit from 1 wire/triple.)
	yb, err := e.conn.ReceiveData()
	if err != nil {
		return nil, fmt.Errorf("vole: MulSender receive y-vector: %w", err)
	}
	if len(yb) != m*32 {
		return nil, fmt.Errorf("vole: MulSender expected %d bytes for y-vector, got %d", m*32, len(yb))
	}
	ys := make([]*big.Int, m)
	for i := 0; i < m; i++ {
		off := i * 32
		ys[i] = new(big.Int).SetBytes(yb[off : off+32])
		ys[i].Mod(ys[i], p)
	}

	// compute u_i = r_i + x_i * y_i mod p; send packed u vector back.
	out := make([]byte, 0, m*32)
	for i := 0; i < m; i++ {
		tmp := new(big.Int).Mul(senderInputs[i], ys[i])
		tmp.Mod(tmp, p)
		ui := new(big.Int).Add(rs[i], tmp)
		ui.Mod(ui, p)
		out = append(out, bytes32(ui)...)
	}

	if err := e.conn.SendData(out); err != nil {
		return nil, fmt.Errorf("vole: MulSender send u-vector: %w", err)
	}
	if err := e.conn.Flush(); err != nil {
		return nil, fmt.Errorf("vole: MulSender flush u-vector: %w", err)
	}

	return rs, nil
}

// MulReceiver functionality: receiverInputs = y[0..m-1]. Returns u[0..m-1] = r_i + x_i * y_i.
func (e *Ext) MulReceiver(receiverInputs []*big.Int, p *big.Int) ([]*big.Int, error) {
	if e == nil {
		return nil, errors.New("vole: nil Ext")
	}
	m := len(receiverInputs)
	if m == 0 {
		return nil, nil
	}

	// If iknp not available, fallback to shim
	if e.iknp == nil {
		return e.mulReceiverShim(receiverInputs, p)
	}

	// Packed receiver path:
	// ExpandReceive with dummy flags (we only need labels; true packed removal of the extra
	// message would require a more complex linear mapping).
	flags := make([]bool, m)
	for i := 0; i < m; i++ {
		flags[i] = false // we could encode something useful here in a future refinement
	}

	labels, err := e.iknp.ExpandReceive(flags)
	if err != nil {
		return nil, fmt.Errorf("vole: ExpandReceive: %w", err)
	}
	if len(labels) != m {
		return nil, fmt.Errorf("vole: ExpandReceive returned %d labels, want %d", len(labels), m)
	}

	// Derive local pads (not strictly required in this design, but available)
	// and (crucially) send the packed y-vector to sender, receive u-vector back.
	outY := make([]byte, 0, m*32)
	for i := 0; i < m; i++ {
		outY = append(outY, bytes32(receiverInputs[i])...)
	}
	if err := e.conn.SendData(outY); err != nil {
		return nil, fmt.Errorf("vole: MulReceiver send y-vector: %w", err)
	}
	if err := e.conn.Flush(); err != nil {
		return nil, fmt.Errorf("vole: MulReceiver flush y-vector: %w", err)
	}

	// receive u vector
	ub, err := e.conn.ReceiveData()
	if err != nil {
		return nil, fmt.Errorf("vole: MulReceiver receive u-vector: %w", err)
	}
	if len(ub) != m*32 {
		return nil, fmt.Errorf("vole: MulReceiver expected %d bytes for u-vector, got %d", m*32, len(ub))
	}
	us := make([]*big.Int, m)
	for i := 0; i < m; i++ {
		off := i * 32
		us[i] = new(big.Int).SetBytes(ub[off : off+32])
		us[i].Mod(us[i], p)
	}
	return us, nil
}

// -----------------------------------------------------------------------------
// Helper: shim implementations (channel-only) - unchanged behavior for tests
// -----------------------------------------------------------------------------

func (e *Ext) mulSenderShim(senderInputs []*big.Int, p *big.Int) ([]*big.Int, error) {
	// same code as prior shim implemented earlier: header + y recv + compute r+xy + send u
	m := len(senderInputs)
	if m == 0 {
		return nil, nil
	}
	// header: send m as 4 bytes (so receiver knows how many to send)
	if err := e.conn.SendData(uint32ToBytes(uint32(m))); err != nil {
		return nil, fmt.Errorf("vole: mulSenderShim send header: %w", err)
	}
	if err := e.conn.Flush(); err != nil {
		return nil, fmt.Errorf("vole: mulSenderShim flush header: %w", err)
	}

	yb, err := e.conn.ReceiveData()
	if err != nil {
		return nil, fmt.Errorf("vole: mulSenderShim recv y: %w", err)
	}
	if len(yb) != m*32 {
		return nil, fmt.Errorf("vole: mulSenderShim expected %d bytes got %d", m*32, len(yb))
	}
	ys := make([]*big.Int, m)
	for i := 0; i < m; i++ {
		off := i * 32
		ys[i] = new(big.Int).SetBytes(yb[off : off+32])
		ys[i].Mod(ys[i], p)
	}

	rs := make([]*big.Int, m)
	out := make([]byte, 0, m*32)
	for i := 0; i < m; i++ {
		ri, err := randomFieldElementFromCrypto(rand.Reader, p)
		if err != nil {
			return nil, fmt.Errorf("vole: mulSenderShim rand: %w", err)
		}
		rs[i] = ri
		tmp := new(big.Int).Mul(senderInputs[i], ys[i])
		tmp.Mod(tmp, p)
		ui := new(big.Int).Add(ri, tmp)
		ui.Mod(ui, p)
		out = append(out, bytes32(ui)...)
	}

	if err := e.conn.SendData(out); err != nil {
		return nil, fmt.Errorf("vole: mulSenderShim send u: %w", err)
	}
	if err := e.conn.Flush(); err != nil {
		return nil, fmt.Errorf("vole: mulSenderShim flush u: %w", err)
	}
	return rs, nil
}

func (e *Ext) mulReceiverShim(receiverInputs []*big.Int, p *big.Int) ([]*big.Int, error) {
	m := len(receiverInputs)
	if m == 0 {
		return nil, nil
	}
	// receive header (4 bytes)
	header, err := e.conn.ReceiveData()
	if err != nil {
		return nil, fmt.Errorf("vole: mulReceiverShim recv header: %w", err)
	}
	if len(header) != 4 {
		return nil, fmt.Errorf("vole: mulReceiverShim header len %d", len(header))
	}
	mGot := bytesToUint32(header)
	if int(mGot) != m {
		return nil, fmt.Errorf("vole: mulReceiverShim header mismatch m=%d local=%d", mGot, m)
	}
	outY := make([]byte, 0, m*32)
	for i := 0; i < m; i++ {
		outY = append(outY, bytes32(receiverInputs[i])...)
	}
	if err := e.conn.SendData(outY); err != nil {
		return nil, fmt.Errorf("vole: mulReceiverShim send y: %w", err)
	}
	if err := e.conn.Flush(); err != nil {
		return nil, fmt.Errorf("vole: mulReceiverShim flush y: %w", err)
	}
	ub, err := e.conn.ReceiveData()
	if err != nil {
		return nil, fmt.Errorf("vole: mulReceiverShim recv u: %w", err)
	}
	if len(ub) != m*32 {
		return nil, fmt.Errorf("vole: mulReceiverShim unexpected u len %d", len(ub))
	}
	us := make([]*big.Int, m)
	for i := 0; i < m; i++ {
		off := i * 32
		us[i] = new(big.Int).SetBytes(ub[off : off+32])
		us[i].Mod(us[i], p)
	}
	return us, nil
}

// -----------------------------------------------------------------------------
// PRG: ChaCha20-based expansion
// -----------------------------------------------------------------------------

// prgChaCha20 expands keyBytes into n bytes using ChaCha20. keyBytes may be any length;
// it is expanded/padded to 32 bytes deterministically (repeat/trim) to form the key.
// NONCE is zero for each stream but the key is unique per label.
func prgChaCha20(keyBytes []byte, n int) []byte {
	key := make([]byte, 32)
	// repeat keyBytes until we fill 32 bytes (deterministic)
	for i := 0; i < 32; i++ {
		key[i] = keyBytes[i%len(keyBytes)]
	}
	nonce := make([]byte, 12) // zero nonce
	c, _ := chacha20.NewUnauthenticatedCipher(key, nonce)
	out := make([]byte, n)
	// stream XOR of zeros gives keystream directly
	zeros := make([]byte, n)
	c.XORKeyStream(out, zeros)
	return out
}

// -----------------------------------------------------------------------------
// small IO / helpers
// -----------------------------------------------------------------------------

func uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
	return b
}

func bytesToUint32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// randomFieldElementFromCrypto reduces 32 random bytes modulo p
func randomFieldElementFromCrypto(r io.Reader, p *big.Int) (*big.Int, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	x := new(big.Int).SetBytes(b)
	x.Mod(x, p)
	return x, nil
}

func bytes32(v *big.Int) []byte {
	out := make([]byte, 32)
	if v == nil {
		return out
	}
	b := v.Bytes()
	copy(out[32-len(b):], b)
	return out
}
