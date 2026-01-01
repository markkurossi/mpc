//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.
//

// Package vole implements a vector-oblivious linear evaluation (VOLE)
// primitive suitable for two-party secure computation protocols such
// as SPDZ.
//
// In a VOLE instance, two parties (sender and receiver) hold private
// vectors x[0..m-1] and y[0..m-1], respectively. They interact to
// obtain correlated, pseudorandom values r[i] and u[i] satisfying:
//
//	u[i] = r[i] + x[i] * y[i] mod p
//
// where p is the prime modulus of the underlying field (e.g., the
// P-256 field).  The sender learns only r[i], and the receiver learns
// only u[i]; neither party learns anything about the other party’s
// inputs beyond what can be inferred from the correlation itself.
//
// This package provides a batched VOLE interface built on top of base
// OT and the IKNP OT extension.  The implementation uses a
// packed-IKNP mode where each VOLE instance consumes only a single
// IKNP wire and expands labels using an AES-CTR PRG. This yields
// large performance improvements compared to bitwise OT
// multiplication.
//
// Typical usage:
//
//	sender, err := vole.NewSender(oti, conn, rand.Reader)
//	if err != nil { ... }
//	rs, _ := sender.Mul(xs, p)
//
//	receiver, err := vole.NewReceiver(oti, conn, rand.Reader)
//	if err != nil { ... }
//	us, _ := receiver.Mul(ys, p)
//
// Here xs and ys are slices of field elements of equal length. The
// sender obtains the r[i] masks, and the receiver obtains u[i] = r[i]
// + x[i]*y[i].
//
// The VOLE construction in this package is intended for semi-honest
// 2PC protocols and is suitable for generating Beaver triple
// cross-terms in SPDZ.  Malicious-secure variants (e.g., with OT
// consistency checks or VOLE sacrifice) are not yet implemented.
package vole
