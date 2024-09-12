//
// Copyright (c) 2024 Markku Rossi
//
// All rights reserved.
//

package utils

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

func Tracef(format string, a ...interface{}) {
	var filename string
	var linenum int

	for skip := 1; ; skip++ {
		pc, file, line, ok := runtime.Caller(skip)
		if !ok {
			break
		}
		f := runtime.FuncForPC(pc)
		if f != nil && strings.HasSuffix(f.Name(), ".errf") {
			continue
		}

		filename = filepath.Base(file)
		linenum = line
		break
	}

	if len(filename) > 0 {
		fmt.Printf("%s:%d: ", filename, linenum)
	}
	msg := fmt.Sprintf(format, a...)
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	fmt.Print(msg)
}
