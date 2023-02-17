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
	const size int = 1024

	wires := make([]Wire, size)
	flags := make([]bool, size)

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

	pipe := NewPipe()

	go func() {
		err := receiver.InitReceiver(pipe)
		if err != nil {
			done <- err
			pipe.Close()
			return
		}
		labels, err := receiver.Receive(flags)
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

		done <- nil
	}()

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

func benchmarkOT(sender, receiver OT, batchSize int, b *testing.B) {
	wires := make([]Wire, batchSize)
	flags := make([]bool, batchSize)

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

	pipe := NewPipe()

	b.ResetTimer()

	go func() {
		for i := 0; i < b.N; i++ {
			err := receiver.InitReceiver(pipe)
			if err != nil {
				done <- err
				pipe.Close()
				return
			}
			labels, err := receiver.Receive(flags)
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
	}()

	for i := 0; i < b.N; i++ {
		err := sender.InitSender(pipe)
		if err != nil {
			b.Fatalf("InitSender: %v", err)
		}
		err = sender.Send(wires)
		if err != nil {
			b.Fatalf("Send: %v", err)
		}
	}

	err := <-done
	if err != nil {
		b.Errorf("receiver failed: %v", err)
	}
}

func BenchmarkOTCO1(b *testing.B) {
	benchmarkOT(NewCO(), NewCO(), 1, b)
}

func BenchmarkOTCO32(b *testing.B) {
	benchmarkOT(NewCO(), NewCO(), 32, b)
}

func BenchmarkOTCO64(b *testing.B) {
	benchmarkOT(NewCO(), NewCO(), 64, b)
}

func BenchmarkOTCO128(b *testing.B) {
	benchmarkOT(NewCO(), NewCO(), 128, b)
}
