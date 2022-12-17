package noise

import (
	"crypto/ed25519"
	"crypto/sha256"
)

// CommitPublicKey commits a public key pk for randomness r as H(pk || s)
func CommitPublicKey(publicKey ed25519.PublicKey, r []byte) []byte {
	input := []byte{}
	input = append(input, []byte(publicKey)...)
	input = append(input, r...)
	res := sha256.Sum256(input)
	return res[:]
}
