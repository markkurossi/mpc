// SHA-1 in Go

package main

import (
	"fmt"

	"github.com/markkurossi/mpc/docs/ref/sha1/crypto/sha1"
)

func main() {
	data := []byte("This page intentionally left blank.")
	fmt.Printf("%x\n", sha1.SumGo(data))
	fmt.Printf("%x\n", sha1.Sum(data))
	fmt.Println()

	data = []byte("----------------------------------------------------------------+!!")
	fmt.Printf("%x\n", sha1.SumGo(data))
	fmt.Printf("%x\n", sha1.Sum(data))
	fmt.Println()

	data = []byte("----------------------------------------------------------------+---------------------------------------------------------------+")
	fmt.Printf("%x\n", sha1.SumGo(data))
	fmt.Printf("%x\n", sha1.Sum(data))
	fmt.Println()

	data = []byte("")
	fmt.Printf("%x\n", sha1.SumGo(data))
	fmt.Printf("%x\n", sha1.Sum(data))
}
