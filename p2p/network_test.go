//
// protocol_test.go
//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package p2p

import (
	"fmt"
	"testing"
)

func TestNetwork(t *testing.T) {
	port := 9000
	numParties := 3
	numConns := 3
	parties := make([]*Network, numParties)
	done := make(chan error)

	nw, err := Create(makeAddr(port), numParties, numConns)
	if err != nil {
		t.Fatal(err)
	}
	parties[0] = nw

	for i := 1; i < numParties; i++ {
		nw, err = Join(makeAddr(port), makeAddr(port+i), i, numConns)
		if err != nil {
			t.Fatal(err)
		}
		go runParty(nw, done)
	}

	// Run leader.
	err = parties[0].Connect()
	if err != nil {
		t.Fatal(err)
	}

	// Wait for parties to complete.
	for i := 1; i < numParties; i++ {
		err = <-done
		if err != nil {
			t.Fatal(err)
		}
	}
}

func runParty(nw *Network, done chan error) {
	done <- partyMain(nw)
}

func partyMain(nw *Network) error {
	return nw.Connect()
}

func makeAddr(port int) string {
	return fmt.Sprintf(":%v", port)
}
