//
// main.go
//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"os"

	"github.com/markkurossi/mpc/ot"
)

func main() {
	m0, _ := ot.NewLabel(rand.Reader)
	m1, _ := ot.NewLabel(rand.Reader)

	sender, err := ot.NewSender(2048)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Sender m0 : %x\n", m0)
	fmt.Printf("  Sender m1 : %x\n", m1)

	receiver, err := ot.NewReceiver(sender.PublicKey())
	if err != nil {
		log.Fatal(err)
	}

	var m0Buf, m1Buf ot.LabelData
	m0Data := m0.Bytes(&m0Buf)
	m1Data := m1.Bytes(&m1Buf)

	sXfer, err := sender.NewTransfer(m0Data, m1Data)
	if err != nil {
		log.Fatal(err)
	}
	rXfer, err := receiver.NewTransfer(0)
	if err != nil {
		log.Fatal(err)
	}

	err = rXfer.ReceiveRandomMessages(sXfer.RandomMessages())
	if err != nil {
		log.Fatal(err)
	}

	sXfer.ReceiveV(rXfer.V())
	err = rXfer.ReceiveMessages(sXfer.Messages())
	if err != nil {
		log.Fatal(err)
	}

	m, bit := rXfer.Message()
	fmt.Printf("Receiver m%d : %x\n", bit, m)

	var ret int
	if bit == 0 {
		ret = bytes.Compare(m0Data[:], m)
	} else {
		ret = bytes.Compare(m1Data[:], m)
	}
	if ret != 0 {
		fmt.Printf("Verify failed!\n")
		os.Exit(1)
	}

}
