package encryption

import (
	"crypto/rand"
	"encoding/hex"
	"golang.org/x/crypto/ed25519"
)

// TODO: Implement this the right way
//Generatekeys - Generate assymetric private/public keys
func GenerateKeys() (privateKey string, publicKey string) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err == nil {
		return hex.EncodeToString(public), hex.EncodeToString(private)
	}
	return "", ""
}

// TODO: Implement this the right way
//Sign - given a private key and data, compute it's signature
func Sign(privateKey string, hash string) string {
	private := []byte(privateKey)
	data := []byte(hash)
	return hex.EncodeToString(ed25519.Sign(private, data))
}

// TODO: Implement this the right way
//Verify - given a public key and a signature,
func Verify(publicKey string, signature string, hash string) bool {
	public := []byte(publicKey)
	sign := []byte(signature)
	data := []byte(hash)
	return ed25519.Verify(public, data, sign)
}
