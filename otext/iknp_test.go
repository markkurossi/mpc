//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.
//

package otext

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

func randomBools(n int) []bool {
	buf := make([]byte, (n+7)/8)
	rand.Read(buf)
	out := make([]bool, n)
	for i := 0; i < n; i++ {
		out[i] = ((buf[i/8] >> uint(i%8)) & 1) == 1
	}
	return out
}

func printRowBytes(prefix string, rows [][]byte, N int) {
	limit := len(rows)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		fmt.Printf("%s row %02d: %s\n", prefix, i, hex.EncodeToString(rows[i]))
	}
	colSample := 8
	if N < colSample {
		colSample = N
	}
	for j := 0; j < colSample; j++ {
		var bits [128]byte
		for i := 0; i < 128 && i < len(rows); i++ {
			byteRow := j / 8
			if byteRow >= len(rows[i]) {
				bits[i] = 0
			} else {
				bits[i] = (rows[i][byteRow] >> uint(j%8)) & 1
			}
		}
		fmt.Printf("%s col %d bits (first 32): %08b\n", prefix, j, bits[:4])
	}
}

func TestIKNP(t *testing.T) {
	c0, c1 := p2p.Pipe()
	oti0 := ot.NewCO(rand.Reader)
	oti1 := ot.NewCO(rand.Reader)

	const N = 200

	var wg sync.WaitGroup
	wg.Add(2)

	var senderWires []ot.Wire
	var recvLabels []ot.Label
	var recvFlags []bool
	var senderExt *IKNPSender
	var receiverExt *IKNPReceiver

	// Sender goroutine
	go func() {
		defer wg.Done()
		oti0.InitSender(c0)
		var err error
		senderExt, err = NewIKNPSender(oti0, c0, rand.Reader)
		if err != nil {
			fatalf("sender setup err: %v", err)
			return
		}
		w, err := senderExt.Expand(N)
		if err != nil {
			fatalf("sender ExpandSend err: %v", err)
			return
		}
		senderWires = w
	}()

	// Receiver goroutine
	go func() {
		defer wg.Done()
		oti1.InitReceiver(c1)
		var err error
		receiverExt, err = NewIKNPReceiver(oti1, c1, rand.Reader)
		if err != nil {
			fatalf("recv setup err: %v", err)
			return
		}
		recvFlags = randomBools(N)
		labels, err := receiverExt.Expand(recvFlags)
		if err != nil {
			fatalf("recv ExpandReceive err: %v", err)
			return
		}
		recvLabels = labels
	}()

	wg.Wait()

	if ferr != nil {
		t.Fatal(ferr)
	}

	//
	// Compare sender wires and receiver outputs
	//
	for j := 0; j < N; j++ {
		var chosen ot.Label
		if recvFlags[j] {
			chosen = senderWires[j].L1
		} else {
			chosen = senderWires[j].L0
		}

		if false {
			fmt.Printf("chosen: %v\n", chosen)
			fmt.Printf(" - L0 : %v\n", senderWires[j].L0)
			fmt.Printf(" - L1 : %v\n", senderWires[j].L1)
		}

		if chosen.Equal(recvLabels[j]) {
			continue
		}

		fmt.Printf("==================================================\n")
		fmt.Printf("Label mismatch at OT index %d\n", j)
		fmt.Printf("recvFlags[%d] = %v\n", j, recvFlags[j])
		fmt.Printf("Sender L0: %s\n", senderWires[j].L0)
		fmt.Printf("Sender L1: %s\n", senderWires[j].L1)
		fmt.Printf("Receiver : %s\n", recvLabels[j])
		fmt.Println()

		// Show sender choice bits
		if senderExt != nil && senderExt.choices != nil {
			fmt.Printf("Sender choice bits (first 64): ")
			for i := 0; i < 64 && i < len(senderExt.choices); i++ {
				if senderExt.choices[i] {
					fmt.Print("1")
				} else {
					fmt.Print("0")
				}
			}
			fmt.Println()
		}

		// Recompute sender rows
		if senderExt != nil && receiverExt != nil {
			Nlocal := N
			rowBytes := (Nlocal + 7) / 8

			// receiver T0/T1
			T0 := make([][]byte, IKNPK)
			T1 := make([][]byte, IKNPK)
			for i := 0; i < IKNPK; i++ {
				T0[i] = make([]byte, rowBytes)
				T1[i] = make([]byte, rowBytes)
				prgAESCTR(receiverExt.seed0[i][:], T0[i])
				prgAESCTR(receiverExt.seed1[i][:], T1[i])
			}

			fmt.Println("Receiver-side T0 sample:")
			printRowBytes(" recv T0", T0, Nlocal)
			fmt.Println("Receiver-side T1 sample:")
			printRowBytes(" recv T1", T1, Nlocal)

			// sender reconstructed rows
			rows := make([][]byte, IKNPK)
			for i := 0; i < IKNPK; i++ {
				rows[i] = make([]byte, rowBytes)
				prgAESCTR(senderExt.seedS[i][:], rows[i])

				if senderExt.choices[i] {
					// apply U = T0 ^ T1
					for b := 0; b < rowBytes; b++ {
						rows[i][b] ^= (T0[i][b] ^ T1[i][b])
					}
				}
			}
			fmt.Println("Sender-side rows sample:")
			printRowBytes(" send rows", rows, Nlocal)
		}

		// Byte-level diff
		var la, lb ot.LabelData
		chosen.GetData(&la)
		recvLabels[j].GetData(&lb)
		fmt.Println("Byte differences (sender Lx vs receiver):")
		for idx := 0; idx < 16; idx++ {
			if la[idx] != lb[idx] {
				fmt.Printf("  byte %02d: sender=%02x recv=%02x\n",
					idx, la[idx], lb[idx])
			}
		}

		t.Fatalf("Label mismatch at %d", j)
	}

	fmt.Println("IKNP test passed with no mismatches.")
}

var m sync.Mutex
var ferr error

func fatalf(format string, a ...interface{}) {
	m.Lock()
	defer m.Unlock()

	if ferr == nil {
		ferr = fmt.Errorf(format, a...)
	}
}
