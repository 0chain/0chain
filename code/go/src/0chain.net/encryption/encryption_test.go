package encryption

import (
	"fmt"
	"testing"
)

var data = "0chain.net rocks"
var expectedHash = "6cb51770083ba34e046bc6c953f9f05b64e16a0956d4e496758b97c9cf5687d5"

func TestHash(t *testing.T) {
	if Hash(data) != expectedHash {
		fmt.Printf("invalid hash")
	} else {
		fmt.Printf("hash successful")
	}
}

func TestGenerateKeys(t *testing.T) {
	publicKey, privateKey := GenerateKeys()
	fmt.Printf("keys: %v,%v\n", privateKey, publicKey)
}

func TestSignAndVerify(t *testing.T) {
	publicKey, privateKey := GenerateKeys()
	signature, err := Sign(privateKey, expectedHash)
	if err != nil {
		fmt.Printf("error signing: %v\n", err)
		return
	}
	fmt.Printf("singing successful\n")
	if ok, err := Verify(publicKey, signature, expectedHash); err != nil || !ok {
		fmt.Printf("Verification failed\n")
	} else {
		fmt.Printf("Signing Verification successful\n")
	}
}
