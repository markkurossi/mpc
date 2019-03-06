//
// main.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/markkurossi/mpc/ot"
)

func main() {
	m0 := []byte{'M', 's', 'g', '0'}
	m1 := []byte{'1', 'g', 's', 'M'}

	sender, err := ot.NewSender(2048, map[int]ot.Wire{
		0: ot.Wire{
			Label0: m0,
			Label1: m1,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Sender m0 : %x\n", m0)
	fmt.Printf("  Sender m1 : %x\n", m1)

	receiver, err := ot.NewReceiver()
	if err != nil {
		log.Fatal(err)
	}

	receiver.ReceivePublicKey(sender.PublicKey())
	err = receiver.ReceiveRandomMessages(sender.RandomMessages())
	if err != nil {
		log.Fatal(err)
	}

	sender.ReceiveV(receiver.V())
	err = receiver.ReceiveMessages(sender.Messages(0))
	if err != nil {
		log.Fatal(err)
	}

	m, bit := receiver.Message()
	fmt.Printf("Receiver m%d : %x\n", bit, m)

	var ret int
	if bit == 0 {
		ret = bytes.Compare(m0, m)
	} else {
		ret = bytes.Compare(m1, m)
	}
	if ret != 0 {
		fmt.Printf("Verify failed!\n")
		os.Exit(1)
	}

}
