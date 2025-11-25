//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.
//

// Package env implements global environment for the MPC system.
package env

import (
	"crypto/rand"
	"io"
)

// Config defines the global system configuration for the MPC system.
// It configures system operation for all MPC modules. Config must not
// be modified after being passed to any MPC module.  It is safe for
// concurrent use by multiple modules as they do not modify it.
type Config struct {
	Rand io.Reader
}

// GetRandom returns the source of entropy for garbling, OT, and other
// cryptography operations.
func (config *Config) GetRandom() io.Reader {
	if config.Rand != nil {
		return config.Rand
	}
	return rand.Reader
}
