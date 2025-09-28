//
// main.go
//
// Copyright (c) 2019-2025 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/markkurossi/mpc/compiler/utils"
)

type pkgPath []string

func (pkg *pkgPath) String() string {
	return fmt.Sprint(*pkg)
}

func (pkg *pkgPath) Set(value string) error {
	for _, v := range strings.Split(value, ":") {
		*pkg = append(*pkg, v)
	}
	return nil
}

var pkgPathFlag pkgPath

func init() {
	flag.Var(&pkgPathFlag, "pkgpath", "colon-separated list of pkg directories")
}

func main() {
	dir := flag.String("dir", ",apidoc",
		"generate documentation to the argument directory")
	flag.Parse()

	log.SetFlags(0)

	if len(flag.Args()) == 0 {
		fmt.Println("no files specified")
		os.Exit(1)
	}

	params := utils.NewParams()
	params.NoCircCompile = true
	params.PkgPath = pkgPathFlag

	doc, err := NewHTMLDoc(*dir)
	if err != nil {
		log.Fatal(err)
	}
	err = documentation(params, flag.Args(), doc)
	if err != nil {
		log.Fatal(err)
	}
}
