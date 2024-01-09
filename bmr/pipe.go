//
// Copyright (c) 2024 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"io"
)

type pipe struct {
	r io.Reader
	w io.Writer
}

func (pipe *pipe) Read(p []byte) (n int, err error) {
	return pipe.r.Read(p)
}

func (pipe *pipe) Write(p []byte) (n int, err error) {
	return pipe.Write(p)
}
