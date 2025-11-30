//
// ot.go
//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.

// Package ot implements oblivious transfer protocols.
package ot

// OT defines the base 1-out-of-2 Oblivious Transfer protocol. The
// sender uses the Send function to send a []Wire array where each
// wire has zero and one Label. The receiver calls Receive with a
// []bool array of selection bits. The higher level protocol must
// ensure the []Wire and []bool array lengths match.
type OT interface {
	// InitSender initializes the OT sender.
	InitSender(io IO) error

	// InitReceiver initializes the OT receiver.
	InitReceiver(io IO) error

	// Send sends the wire labels with OT.
	Send(wires []Wire) error

	// Receive receives the wire labels with OT based on the flag values.
	Receive(flags []bool, result []Label) error
}
