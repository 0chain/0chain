package encryption

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"io"

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

/*ReadKeys - reads a publicKey and a privateKey from a Reader.
They are assumed to be in two separate lines one followed by the other*/
func ReadKeys(reader io.Reader) (publicKey string, privateKey string) {
	scanner := bufio.NewScanner(reader)
	scanner.Scan()
	publicKey = scanner.Text()
	scanner.Scan()
	privateKey = scanner.Text()
	return publicKey, privateKey
}

/*SignerVerifier - an interface that can sign a hash and verify a signature and hash */
type SignerVerifier interface {
	Sign(hash string) (string, error)
	Verify(signature string, hash string) (bool, error)
}

//Sign - given a private key and data, compute it's signature
func Sign(privateKey string, hash interface{}) (string, error) {
	private, err := hex.DecodeString(privateKey)
	if err != nil {
		return "", err
	}
	var rawHash []byte
	switch hashImpl := hash.(type) {
	case []byte:
		rawHash = hashImpl
	case string:
		decoded, err := hex.DecodeString(hashImpl)
		if err != nil {
			return "", err
		}
		rawHash = decoded
	default:
		panic("unknown hash type")
	}

	return hex.EncodeToString(ed25519.Sign(private, rawHash)), nil
}

//Verify - given a public key and a signature and the hash used to create the signature, verify the signature
func Verify(publicKey interface{}, signature string, hash string) (bool, error) {
	var public []byte
	switch publicImpl := publicKey.(type) {
	case []byte:
		public = publicImpl
	case string:
		decoded, err := hex.DecodeString(publicImpl)
		if err != nil {
			return false, err
		}
		public = decoded
	default:
		panic("unknown public key type")
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
