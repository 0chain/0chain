package encryption

import (
	"crypto/rand"
	"encoding/hex"

	"golang.org/x/crypto/ed25519"
)

//GenerateKeys - Generate assymetric private/public keys
func GenerateKeys() (publicKey string, privateKey string) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", ""
	}
	return hex.EncodeToString(public), hex.EncodeToString(private)
}

//Sign - given a private key and data, compute it's signature
func Sign(privateKey string, hash string) (string, error) {
	private, err := hex.DecodeString(privateKey)
	if err != nil {
		return "", err
	}
	data, err := hex.DecodeString(hash)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ed25519.Sign(private, data)), nil
}

//Verify - given a public key and a signature,
func Verify(publicKey string, signature string, hash string) (bool, error) {
	public, err := hex.DecodeString(publicKey)
	if err != nil {
		return false, err
	}
	sign, err := hex.DecodeString(signature)
	if err != nil {
		return false, err
	}
	data, err := hex.DecodeString(hash)
	if err != nil {
		return false, err
	}
	return ed25519.Verify(public, data, sign), nil
}
