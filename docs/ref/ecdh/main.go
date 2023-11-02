// ecdh in Go

package main

import (
	"crypto/rand"
	"fmt"
	"log"

	"github.com/markkurossi/mpc/docs/ref/ecdh/curve25519"
)

func main() {

	var privateKey [32]byte

	_, err := rand.Read(privateKey[:])
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < len(privateKey); i++ {
		privateKey[i] = byte((i % 8) + 1)
	}

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	fmt.Printf("Alice priv  : %x\n", privateKey)
	fmt.Printf("Alice pub   : %x\n", publicKey)

	var privateKey2 [32]byte
	_, err = rand.Read(privateKey2[:])
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < len(privateKey2); i++ {
		privateKey2[i] = byte((i % 16) + 1)
	}

	var publicKey2 [32]byte
	curve25519.ScalarBaseMult(&publicKey2, &privateKey2)

	var out1, out2 [32]byte

	fmt.Printf("Bob priv    : %x\n", privateKey2)
	fmt.Printf("Bob pub     : %x\n", publicKey2)

	fmt.Printf(" ScalarMult(0x%x,\n\t    0x%x)\n", privateKey, publicKey2)
	curve25519.ScalarMult(&out1, &privateKey, &publicKey2)
	curve25519.ScalarMult(&out2, &privateKey2, &publicKey)

	fmt.Printf("Alice shared: %x\n", out1)
	fmt.Printf("Bob shared  : %x\n", out2)

}
