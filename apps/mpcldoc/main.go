//
// main.go
//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	dir := flag.String("dir", ",apidoc",
		"generate documentation to the argument directory")
	flag.Parse()

	log.SetFlags(0)

	if len(flag.Args()) == 0 {
		fmt.Println("no files specified")
		os.Exit(1)
	}

	doc, err := NewHTMLDoc(*dir)
	if err != nil {
		log.Fatal(err)
	}
	err = documentation(flag.Args(), doc)
	if err != nil {
		log.Fatal(err)
	}
}
