package encryption

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"io"

	"golang.org/x/crypto/ed25519"
)

//GenerateKeys - Generate assymetric private/public keys
func GenerateKeys() (publicKey string, privateKey string, err error) {
	public, private, err := GenerateKeysBytes()
	if err != nil {
		return "", "", err
	}
	return hex.EncodeToString(public), hex.EncodeToString(private), nil
}

//GenerateKeysBytes - Generate assymetric private/public keys
func GenerateKeysBytes() ([]byte, []byte, error) {
	return ed25519.GenerateKey(rand.Reader)
}

/*ReadKeys - reads a publicKey and a privateKey from a Reader.
They are assumed to be in two separate lines one followed by the other*/
func ReadKeys(reader io.Reader) (success bool, publicKey string, privateKey string) {
	publicKey = ""
	privateKey = ""
	scanner := bufio.NewScanner(reader)
	result := scanner.Scan()
	if !result {
		return false, publicKey, privateKey
	}
	publicKey = scanner.Text()
	result = scanner.Scan()
	if !result {
		return false, publicKey, privateKey
	}
	privateKey = scanner.Text()
	return true, publicKey, privateKey
}

//Sign - given a private key and data, compute it's signature
func Sign(privateKey interface{}, hash interface{}) (string, error) {
	return signED25519(privateKey, hash)
}

//Verify - given a public key and a signature and the hash used to create the signature, verify the signature
func Verify(publicKey interface{}, signature string, hash string) (bool, error) {
	return verifyED25519(publicKey, signature, hash)
}

// GetClientIDFromPublicKey - given the PK of the provider, return the its operational walletID i.e. the key used to sign the txns
func GetClientIDFromPublicKey(pk string) (string, error) {
	publicKeyBytes, err := hex.DecodeString(pk)
	if err != nil {
		return "", err
	}
	operationalClientID := Hash(publicKeyBytes)
	return operationalClientID, nil
}