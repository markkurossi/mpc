//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package utils

import (
	"fmt"
	"io"
)

type Logger struct {
	input string
	out   io.Writer
}

func NewLogger(input string, out io.Writer) *Logger {
	return &Logger{
		input: input,
		out:   out,
	}
}

func (l *Logger) Errorf(loc Point, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if loc.Line == 0 {
		fmt.Fprintf(l.out, "%s: %s", l.input, msg)
	} else {
		fmt.Fprintf(l.out, "%s:%s: %s", l.input, loc, msg)
	}
}

func (l *Logger) Warningf(loc Point, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if loc.Line == 0 {
		fmt.Fprintf(l.out, "%s: warning: %s", l.input, msg)
	} else {
		fmt.Fprintf(l.out, "%s:%s: warning: %s", l.input, loc, msg)
	}
}
