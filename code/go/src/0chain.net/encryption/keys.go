package encryption

import (
	"fmt"
	"math/rand"
)

var r = rand.New(rand.NewSource(99))

// TODO: Implement this the right way
//Generatekeys - Generate assymetric private/public keys
func GenerateKeys() (privateKey string, publicKey string) {
	id := r.Int63()
	k1 := fmt.Sprintf("0chain.net key %v", id)
	id = r.Int63()
	k2 := fmt.Sprintf("0chain.net key %v", id)
	return Hash(k1), Hash(k2)
}

// TODO: Implement this the right way
//Sign - given a private key and data, compute it's signature
func Sign(privateKey string, hash string) string {
	return Hash(hash)
}

// TODO: Implement this the right way
//Verify - given a public key and a signature,
func Verify(publicKey string, signature string, hash string) bool {
	return Hash(hash) == signature
}
