package encryption

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var ErrKeyRead = errors.New("error reading the keys")
var ErrInvalidSignatureScheme = errors.New("invalid signature scheme")

//SignatureScheme - an encryption scheme for signing and verifying messages
type SignatureScheme interface {
	GenerateKeys() error

	ReadKeys(reader io.Reader) error
	WriteKeys(writer io.Writer) error

	SetPublicKey(publicKey string) error
	GetPublicKey() string

	Sign(hash interface{}) (string, error)
	Verify(signature string, hash string) (bool, error)
}

//AggregateSignatureScheme - a signature scheme that can aggregate individual signatures
type AggregateSignatureScheme interface {
	Aggregate(ss SignatureScheme, idx int, signature string, hash string) error
	Verify() (bool, error)
}

//GetSignatureScheme - given the name, return a signature scheme
func GetSignatureScheme(sigScheme string) SignatureScheme {
	switch sigScheme {
	case "ed25519":
		return NewED25519Scheme()

	case "bls0chain":
		return NewBLS0ChainScheme()
	default:
		panic(fmt.Sprintf("unknown signature scheme: %v", sigScheme))
	}
}

//GetAggregateSignatureScheme - get an aggregate signature scheme
func GetAggregateSignatureScheme(sigScheme string, total int, batchSize int) AggregateSignatureScheme {
	switch sigScheme {
	case "ed25519":
		return nil
	case "bls0chain":
		return NewBLS0ChainAggregateSignature(total, batchSize)
	default:
		panic(fmt.Sprintf("unknown signature scheme: %v", sigScheme))
	}
}

//GetRawHash - given a hash interface (raw hash, hex string), return the raw hash
func GetRawHash(hash interface{}) ([]byte, error) {
	switch hashImpl := hash.(type) {
	case []byte:
		return hashImpl, nil
	case string:
		decoded, err := hex.DecodeString(hashImpl)
		if err != nil {
			return nil, err
		}
		return decoded, nil
	default:
		panic("unknown hash type")
	}
}
