// ed25519 in Go

package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/markkurossi/mpc/docs/ref/ed25519/edwards25519"
)

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		usage()
		os.Exit(1)
	}
	var err error

	switch flag.Args()[0] {
	case "keygen":
		err = keygen()

	case "sign":
		err = sign()

	case "verify":
		if len(flag.Args()) != 4 {
			usage()
			os.Exit(1)
		}
		err = verify(flag.Args()[1], flag.Args()[2], flag.Args()[3])

	default:
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Printf("%s failed: %s\n", flag.Args()[0], err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf("usage: ed25519 {keygen,sign,verify PUB MSG SIG}\n")
}

const (
	// PublicKeySize is the size, in bytes, of public keys as used in
	// this package.
	PublicKeySize = 32
	// PrivateKeySize is the size, in bytes, of private keys as used
	// in this package.
	PrivateKeySize = 64
	// SignatureSize is the size, in bytes, of signatures generated
	// and verified by this package.
	SignatureSize = 64
	// SeedSize is the size, in bytes, of private key seeds. These are
	// the private key representations used by RFC 8032.
	SeedSize = 32
)

// PrivateKey defines the Ed25519 private key.
type PrivateKey [64]byte

// PublicKey defines the Ed25519 public key.
type PublicKey [32]byte

const (
	ArgSeedE int = iota
	ArgSeedG
	ArgSplitE
	ArgSplitG
	ArgMaskE
	ArgMaskG
	ArgPrivateKey
	ArgPublicKey
	ArgMessage
)

var argStrings = []string{
	// Seeds E and G
	"784db0ec4ca0cf5338249e6a09139109366dca1fac2838e5f0e5a46f0e191bae",
	"57c0e59c20ac7d75ef7e3188fdd7f5876abee1cab394af8125acaca9760bb54c",

	// Splits E and G
	"d0da45d3c99e756da831d1e7d696eae3fa9fe39d3b1b2618c7ff997d17777989b5cf415b114298c8b10bed0f0eff118e43ab606ab01143151dff89171307dffa",
	"76b42e6292f4a3dc339d208481abeb9a24e08127c7cd8dbde62abcddc0c0e6f7a0f740e756b44dae137f0e7ff8eae0ceb1a962c130fdcbe8cbee3e31ab55b8dc",

	// Masks E and G
	"44bf09357e19b1f96f9cf6d9e7d25a0e8dd62d6e0d4bba2bec4c59983c7dc84d1486677b6d8837746cd948c881913c36faeaee08e8309afac58be4757a1c544e",
	"eb83eb1f5203f5b752c96264a21ff4a27fa60cf2313f5f53c3fa96e0b52a2814b786e43a3af64b66291b5b29f432cb8d5a930e31f4e6f072a6d33b861b5b5f13",

	// Private and public key.
	"2f8d55706c0cb226d75aafe2f4c4648e5cd32bd51fbc9764d54908c67812aee28ae64963506002e267a59665e9a2e6f9348cc159be53747894478e182ece9fcb",
	"8ae64963506002e267a59665e9a2e6f9348cc159be53747894478e182ece9fcb",

	// Message
	"4d61726b6b7520526f737369203c6d747240696b692e66693e2068747470733a2f2f7777772e6d61726b6b75726f7373692e636f6d2f",
}

var args [][]byte

func init() {
	for _, arg := range argStrings {
		data, err := hex.DecodeString(arg)
		if err != nil {
			panic("hex decode: " + err.Error())
		}

		args = append(args, data)
	}
	fmt.Printf("Message: %s\n", string(args[ArgMessage]))
}

func keygen() error {
	var seed [32]byte

	for idx := range seed {
		seed[idx] = args[ArgSeedE][idx] ^ args[ArgSeedG][idx]
	}
	pub, priv := NewKeyFromSeed(seed)

	var shareG [64]byte
	for idx := range shareG {
		shareG[idx] = args[ArgSplitE][idx] ^ args[ArgSplitG][idx]
	}

	var shareE [64]byte
	for idx := range shareE {
		shareE[idx] = priv[idx] ^ shareG[idx]
	}

	var maskedG [64]byte
	for idx := range maskedG {
		maskedG[idx] = shareG[idx] ^ args[ArgMaskG][idx]
	}
	var maskedE [64]byte
	for idx := range maskedE {
		maskedE[idx] = shareE[idx] ^ args[ArgMaskE][idx]
	}

	fmt.Printf("seed   : %x\n", seed)
	fmt.Printf("pub    : %x\n", pub)
	fmt.Printf("priv   : %x\n", priv)
	fmt.Printf("shareG : %x\n", shareG)
	fmt.Printf("shareE : %x\n", shareE)
	fmt.Printf("maskedG: %x\n", maskedG)
	fmt.Printf("maskedE: %x\n", maskedE)

	return nil
}

func sign() error {
	var priv PrivateKey
	var msg [64]byte

	copy(priv[:], args[ArgPrivateKey])
	copy(msg[:], args[ArgMessage])

	signature := Sign(priv, msg[:])
	fmt.Printf("signature: %x\n", signature)
	return nil
}

func verify(pubStr, msgStr, sigStr string) error {
	pub, err := hex.DecodeString(pubStr)
	if err != nil {
		return err
	}
	msg, err := hex.DecodeString(msgStr)
	if err != nil {
		return err
	}
	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		return err
	}
	if !Verify(pub, msg, sig) {
		return fmt.Errorf("signature verification failed")
	}
	fmt.Printf("signature verification success\n")
	return nil
}

// NewKeyFromSeed calculates a private key and a public key from a
// seed. RFC 8032's private keys correspond to seeds in this package.
func NewKeyFromSeed(seed [SeedSize]byte) (PublicKey, PrivateKey) {
	digest := sha512.Sum512(seed[:])
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	var A edwards25519.ExtendedGroupElement
	var hBytes [32]byte

	copy(hBytes[:], digest[:])

	edwards25519.GeScalarMultBase(&A, &hBytes)
	var publicKeyBytes [32]byte
	A.ToBytes(&publicKeyBytes)

	var privateKey [64]byte
	copy(privateKey[:], seed[:])
	copy(privateKey[32:], publicKeyBytes[:])

	return publicKeyBytes, privateKey
}

// Sign signs the message with privateKey and returns the signature.
func Sign(privateKey PrivateKey, message []byte) []byte {

	fmt.Printf("priv     : %x\n", privateKey)
	fmt.Printf("message  : %x\n", message)

	digest1 := sha512.Sum512(privateKey[0:32])

	var expandedSecretKey [32]byte
	copy(expandedSecretKey[:], digest1[:])
	expandedSecretKey[0] &= 248
	expandedSecretKey[31] &= 63
	expandedSecretKey[31] |= 64

	buf := make([]byte, 32+len(message))
	copy(buf[0:], digest1[32:])
	copy(buf[32:], message[:])
	fmt.Printf("buf: %x\n", buf)
	messageDigest := sha512.Sum512(buf)

	fmt.Printf("messageDigest: %x\n", messageDigest)

	var messageDigestReduced [32]byte
	edwards25519.ScReduce(&messageDigestReduced, &messageDigest)

	fmt.Printf("messageDigestReduced: %x\n", messageDigestReduced)

	var R edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&R, &messageDigestReduced)

	var encodedR [32]byte
	R.ToBytes(&encodedR)

	buf2 := make([]byte, 64+len(message))
	copy(buf2[0:], encodedR[:])
	copy(buf2[32:], privateKey[32:])
	copy(buf2[64:], message[:])
	hramDigest := sha512.Sum512(buf2)

	var hramDigestReduced [32]byte
	edwards25519.ScReduce(&hramDigestReduced, &hramDigest)

	var s [32]byte
	edwards25519.ScMulAdd(&s, &hramDigestReduced, &expandedSecretKey,
		&messageDigestReduced)

	var signature [SignatureSize]byte
	copy(signature[0:], encodedR[:])
	copy(signature[32:], s[:])

	return signature[:]
}

// Verify reports whether sig is a valid signature of message by publicKey. It
// will panic if len(publicKey) is not PublicKeySize.
func Verify(publicKey, message, sig []byte) bool {
	if l := len(publicKey); l != PublicKeySize {
		panic("ed25519: bad public key length: " + strconv.Itoa(l))
	}

	if len(sig) != SignatureSize || sig[63]&224 != 0 {
		return false
	}

	var A edwards25519.ExtendedGroupElement
	var publicKeyBytes [32]byte
	copy(publicKeyBytes[:], publicKey)
	if !A.FromBytes(&publicKeyBytes) {
		return false
	}
	edwards25519.FeNeg(&A.X, &A.X)
	edwards25519.FeNeg(&A.T, &A.T)

	h := sha512.New()
	h.Write(sig[:32])
	h.Write(publicKey[:])
	h.Write(message)
	var digest [64]byte
	h.Sum(digest[:0])

	var hReduced [32]byte
	edwards25519.ScReduce(&hReduced, &digest)

	var R edwards25519.ProjectiveGroupElement
	var s [32]byte
	copy(s[:], sig[32:])

	// https://tools.ietf.org/html/rfc8032#section-5.1.7 requires that s be in
	// the range [0, order) in order to prevent signature malleability.
	if !edwards25519.ScMinimal(&s) {
		return false
	}

	edwards25519.GeDoubleScalarMultVartime(&R, &hReduced, &A, &s)

	var checkR [32]byte
	R.ToBytes(&checkR)
	return bytes.Equal(sig[:32], checkR[:])
}
