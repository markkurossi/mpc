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
	"sync"
	"testing"
)

func TestNetwork(t *testing.T) {
	port := 9000
	numParties := 5
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

	var wg sync.WaitGroup

	self := parties[0].Self

	for _, peer := range parties[0].Peers {
		if peer.ID == self.ID {
			continue
		}
		for i, conn := range peer.Conns {
			if self.ID < peer.ID {
				wg.Go(func() {
					ping(conn, self, peer, i)
				})
			} else {
				wg.Go(func() {
					pong(conn, self, peer, i)
				})
			}
		}
	}

	wg.Wait()
}

func runParty(nw *Network, done chan error) {
	if err := nw.Connect(); err != nil {
		done <- err
		return
	}
	done <- nil

	self := nw.Self

	var wg sync.WaitGroup

	for _, peer := range nw.Peers {
		if peer.ID == self.ID {
			continue
		}
		for i, conn := range peer.Conns {
			if self.ID < peer.ID {
				wg.Go(func() {
					ping(conn, self, peer, i)
				})
			} else {
				wg.Go(func() {
					pong(conn, self, peer, i)
				})
			}
		}
	}

	wg.Wait()
}

func ping(conn *Conn, from, to *Peer, connID int) {
	req := "ping"
	if err := conn.SendString("ping"); err != nil {
		panic(err)
	}
	if err := conn.Flush(); err != nil {
		panic(err)
	}
	resp, err := conn.ReceiveString()
	if err != nil {
		panic(err)
	}
	if resp != req {
		panic(resp)
	}
}

func pong(conn *Conn, from, to *Peer, connID int) {
	msg, err := conn.ReceiveString()
	if err != nil {
		panic(err)
	}
	fmt.Printf("pong: from=%v, to=%v, conn=%v\n", from, to, connID)
	if err := conn.SendString(msg); err != nil {
		panic(err)
	}
	if err := conn.Flush(); err != nil {
		panic(err)
	}
}

func makeAddr(port int) string {
	return fmt.Sprintf(":%v", port)
}
