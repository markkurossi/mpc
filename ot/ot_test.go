//
// ot_test.go
//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"crypto/rand"
	"fmt"
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
	testOT(NewCO(), NewCO(), t)
}

func TestOTRSA(t *testing.T) {
	testOT(NewRSA(2048), NewRSA(2048), t)
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

	err := sender.InitSender(pipe)
	if err != nil {
		b.Fatalf("InitSender: %v", err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = sender.Send(wires)
		if err != nil {
			b.Fatalf("Send: %v", err)
		}
	}

	err = <-done
	if err != nil {
		b.Errorf("receiver failed: %v", err)
	}
}

func BenchmarkOTCO_1(b *testing.B) {
	benchmarkOT(NewCO(), NewCO(), 1, b)
}

func BenchmarkOTCO_8(b *testing.B) {
	benchmarkOT(NewCO(), NewCO(), 8, b)
}

func BenchmarkOTCO_16(b *testing.B) {
	benchmarkOT(NewCO(), NewCO(), 16, b)
}

func BenchmarkOTCO_32(b *testing.B) {
	benchmarkOT(NewCO(), NewCO(), 32, b)
}

func BenchmarkOTCO_64(b *testing.B) {
	benchmarkOT(NewCO(), NewCO(), 64, b)
}

func benchmarkOTRSA(keySize, batchSize int, b *testing.B) {
	benchmarkOT(NewRSA(keySize), NewRSA(keySize), batchSize, b)
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
