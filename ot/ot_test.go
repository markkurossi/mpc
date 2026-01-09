//
// ot_test.go
//
// Copyright (c) 2023-2026 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	mrand "math/rand"
	"testing"
)

func testOT(sender, receiver OT, t *testing.T) {
	const size int = 64

	wires := make([]Wire, size)
	flags := make([]bool, size)
	labels := make([]Label, size)

	done := make(chan error)

	for i := 0; i < len(wires); i++ {
		var data LabelData
		_, err := rand.Read(data[:])
		if err != nil {
			t.Fatal(err)
		}
		wires[i].L0.SetData(&data)

		_, err = rand.Read(data[:])
		if err != nil {
			t.Fatal(err)
		}
		wires[i].L1.SetData(&data)

		flags[i] = i%2 == 0
	}

	pipe, rPipe := NewPipe()

	go func(pipe *Pipe) {
		err := receiver.InitReceiver(pipe)
		if err != nil {
			pipe.Close()
			pipe.Drain()
			done <- err
			return
		}
		err = receiver.Receive(flags, labels)
		if err != nil {
			pipe.Close()
			pipe.Drain()
			done <- err
			return
		}
		for i := 0; i < len(flags); i++ {
			var expected Label
			if flags[i] {
				expected = wires[i].L1
			} else {
				expected = wires[i].L0
			}
			if !labels[i].Equal(expected) {
				err := fmt.Errorf("label %d mismatch %v %v,%v", i,
					labels[i], wires[i].L0, wires[i].L1)
				pipe.Close()
				done <- err
				return
			}
		}

		done <- nil
	}(rPipe)

	err := sender.InitSender(pipe)
	if err != nil {
		t.Fatalf("InitSender: %v", err)
	}
	err = sender.Send(wires)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	err = <-done
	if err != nil {
		t.Errorf("receiver failed: %v", err)
	}
}

func TestOTCO(t *testing.T) {
	testOT(NewCO(rand.Reader), NewCO(rand.Reader), t)
}

func TestOTRSA(t *testing.T) {
	testOT(NewRSA(rand.Reader, 2048), NewRSA(rand.Reader, 2048), t)
}

func benchmarkOT(sender, receiver OT, batchSize int, b *testing.B) {
	wires := make([]Wire, batchSize)
	flags := make([]bool, batchSize)
	labels := make([]Label, batchSize)

	done := make(chan error)

	for i := 0; i < len(wires); i++ {
		var data LabelData
		_, err := rand.Read(data[:])
		if err != nil {
			b.Fatal(err)
		}
		wires[i].L0.SetData(&data)

		_, err = rand.Read(data[:])
		if err != nil {
			b.Fatal(err)
		}
		wires[i].L1.SetData(&data)

		flags[i] = i%2 == 0
	}

	pipe, rPipe := NewPipe()

	go func(pipe *Pipe) {
		err := receiver.InitReceiver(pipe)
		if err != nil {
			done <- err
			pipe.Close()
			return
		}
		for i := 0; i < b.N; i++ {
			err = receiver.Receive(flags, labels)
			if err != nil {
				done <- err
				pipe.Close()
				return
			}
			for i := 0; i < len(flags); i++ {
				var expected Label
				if flags[i] {
					expected = wires[i].L1
				} else {
					expected = wires[i].L0
				}
				if !labels[i].Equal(expected) {
					err := fmt.Errorf("label %d mismatch %v %v,%v", i,
						labels[i], wires[i].L0, wires[i].L1)
					done <- err
					pipe.Close()
					return
				}
			}
		}

		done <- nil
	}(rPipe)

	go func(pipe *Pipe) {
		err := sender.InitSender(pipe)
		if err != nil {
			done <- err
			return
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err = sender.Send(wires)
			if err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}(pipe)

	for i := 0; i < 2; i++ {
		err := <-done
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOTCO_1(b *testing.B) {
	benchmarkOT(NewCO(rand.Reader), NewCO(rand.Reader), 1, b)
}

func BenchmarkOTCO_8(b *testing.B) {
	benchmarkOT(NewCO(rand.Reader), NewCO(rand.Reader), 8, b)
}

func BenchmarkOTCO_16(b *testing.B) {
	benchmarkOT(NewCO(rand.Reader), NewCO(rand.Reader), 16, b)
}

func BenchmarkOTCO_32(b *testing.B) {
	benchmarkOT(NewCO(rand.Reader), NewCO(rand.Reader), 32, b)
}

func BenchmarkOTCO_64(b *testing.B) {
	benchmarkOT(NewCO(rand.Reader), NewCO(rand.Reader), 64, b)
}

// readLabelFromReader deterministically fills a label using the reader's bytes.
func readLabelFromReader(reader io.Reader) (Label, error) {
	var data LabelData
	if _, err := io.ReadFull(reader, data[:]); err != nil {
		return Label{}, err
	}
	var label Label
	label.SetData(&data)
	return label, nil
}

// deterministicReader is a math/rand-backed reader for reproducible tests only.
type deterministicReader struct {
	src *mrand.Rand
}

// newDeterministicReader creates a deterministicReader seeded from the string.
func newDeterministicReader(seed string) *deterministicReader {
	sum := sha256.Sum256([]byte(seed))
	src := mrand.NewSource(int64(binary.BigEndian.Uint64(sum[:8])))
	return &deterministicReader{src: mrand.New(src)}
}

// Read fills the buffer with pseudo-random bytes (not cryptographically safe).
func (r *deterministicReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(r.src.Intn(256))
	}
	return len(p), nil
}

// TestCODeterministicTranscript locks the IO CO OT to a known hash.
func TestCODeterministicTranscript(t *testing.T) {
	const wiresCount = 8
	wires := make([]Wire, wiresCount)
	wireRand := newDeterministicReader("io-co-wires")
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
	flags := []bool{false, true, false, true, true, false, true, false}

	type transcriptResult struct {
		labels []Label
		err    error
	}
	resultCh := make(chan transcriptResult, 1)

	senderPipe, receiverPipe := NewPipe()

	go func() {
		receiver := NewCO(newDeterministicReader("io-receiver"))
		if err := receiver.InitReceiver(receiverPipe); err != nil {
			resultCh <- transcriptResult{err: err}
			return
		}
		labels := make([]Label, wiresCount)
		if err := receiver.Receive(flags, labels); err != nil {
			resultCh <- transcriptResult{err: err}
			return
		}
		for idx, bit := range flags {
			expected := wires[idx].L0
			if bit {
				expected = wires[idx].L1
			}
			if !labels[idx].Equal(expected) {
				resultCh <- transcriptResult{
					err: fmt.Errorf("label %d mismatch", idx),
				}
				return
			}
		}
		copyLabels := make([]Label, len(labels))
		copy(copyLabels, labels)
		resultCh <- transcriptResult{labels: copyLabels}
	}()

	sender := NewCO(newDeterministicReader("io-sender"))
	if err := sender.InitSender(senderPipe); err != nil {
		t.Fatalf("InitSender: %v", err)
	}
	if err := sender.Send(wires); err != nil {
		t.Fatalf("Send: %v", err)
	}

	result := <-resultCh
	if result.err != nil {
		t.Fatalf("receiver failed: %v", result.err)
	}

	var buf bytes.Buffer
	var tmp LabelData
	for _, wire := range wires {
		wire.L0.GetData(&tmp)
		buf.Write(tmp[:])
		wire.L1.GetData(&tmp)
		buf.Write(tmp[:])
	}
	for _, flag := range flags {
		if flag {
			buf.WriteByte(1)
		} else {
			buf.WriteByte(0)
		}
	}
	for _, label := range result.labels {
		label.GetData(&tmp)
		buf.Write(tmp[:])
	}
	// coTranscriptHash records the deterministic CO transcript digest.
	const coTranscriptHash = "665c4a1093bb2792f09808a5113dcc57c13aae7ebb65cf041faeace305fca55e"
	hash := fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))
	if hash != coTranscriptHash {
		t.Fatalf("transcript hash mismatch: got %s want %s", hash, coTranscriptHash)
	}
}

func benchmarkOTRSA(keySize, batchSize int, b *testing.B) {
	benchmarkOT(NewRSA(rand.Reader, keySize), NewRSA(rand.Reader, keySize),
		batchSize, b)
}

func BenchmarkOTRSA_2048_1(b *testing.B) {
	benchmarkOTRSA(2048, 1, b)
}

func BenchmarkOTRSA_2048_8(b *testing.B) {
	benchmarkOTRSA(2048, 8, b)
}

func BenchmarkOTRSA_2048_64(b *testing.B) {
	benchmarkOTRSA(2048, 64, b)
}

func benchmarkOTCOT(batchSize int, b *testing.B) {
	benchmarkOT(NewCOT(NewCO(rand.Reader), rand.Reader, false, false),
		NewCOT(NewCO(rand.Reader), rand.Reader, false, false),
		batchSize, b)
}

func BenchmarkOTCOT_1(b *testing.B) {
	benchmarkOTCOT(1, b)
}
func BenchmarkOTCOT_8(b *testing.B) {
	benchmarkOTCOT(8, b)
}
func BenchmarkOTCOT_16(b *testing.B) {
	benchmarkOTCOT(16, b)
}
func BenchmarkOTCOT_32(b *testing.B) {
	benchmarkOTCOT(32, b)
}
func BenchmarkOTCOT_64(b *testing.B) {
	benchmarkOTCOT(64, b)
}
func BenchmarkOTCOT_128(b *testing.B) {
	benchmarkOTCOT(128, b)
}
func BenchmarkOTCOT_256(b *testing.B) {
	benchmarkOTCOT(256, b)
}
func BenchmarkOTCOT_512(b *testing.B) {
	benchmarkOTCOT(512, b)
}
