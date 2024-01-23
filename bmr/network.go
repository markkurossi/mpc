//
// Copyright (c) 2024 Markku Rossi
//
// All rights reserved.
//

package bmr

// Operand defines protocol operands.
//
//go:generate stringer -type=Operand -trimprefix=Op
type Operand byte

// Network protocol messages.
const (
	OpInit Operand = iota
	OpFxLambda
	OpFxR
)
