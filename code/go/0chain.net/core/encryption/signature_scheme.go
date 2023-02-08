package encryption

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

const (
	SignatureSchemeEd25519   = string("ed25519")
	SignatureSchemeBls0chain = string("bls0chain")
)

var ErrKeyRead = errors.New("error reading the keys")
var ErrInvalidSignatureScheme = errors.New("invalid signature scheme")

// SignatureScheme - an encryption scheme for signing and verifying messages
type SignatureScheme interface {
	GenerateKeys() error

	ReadKeys(reader io.Reader) error
	WriteKeys(writer io.Writer) error

	SetPublicKey(publicKey string) error
	GetPublicKey() string

	Sign(hash interface{}) (string, error)
	Verify(signature string, hash string) (bool, error)
}

// AggregateSignatureScheme - a signature scheme that can aggregate individual signatures
type AggregateSignatureScheme interface {
	Aggregate(ss SignatureScheme, idx int, signature string, hash string) error
	Verify() (bool, error)
}

type ThresholdSignatureScheme interface {
	SignatureScheme

	SetID(id string) error
	GetID() string
}

type ReconstructSignatureScheme interface {
	Add(tss ThresholdSignatureScheme, signature string) error
	Reconstruct() (string, error)
}

// IsValidSignatureScheme - whether a signature scheme exists
func IsValidSignatureScheme(sigScheme string) bool {
	switch sigScheme {
	case SignatureSchemeEd25519:
		return true
	case SignatureSchemeBls0chain:
		return true
	default:
		return false
	}
}

// GetSignatureScheme - given the name, return a signature scheme
func GetSignatureScheme(sigScheme string) SignatureScheme {
	switch sigScheme {
	case SignatureSchemeEd25519:
		return NewED25519Scheme()
	case SignatureSchemeBls0chain:
		return NewBLS0ChainScheme()
	default:
		panic(fmt.Sprintf("unknown signature scheme: %v", sigScheme))
	}
}

// IsValidAggregateSignatureScheme - whether an aggregate signature scheme exists
func IsValidAggregateSignatureScheme(sigScheme string) bool {
	switch sigScheme {
	case SignatureSchemeEd25519:
		return false
	case SignatureSchemeBls0chain:
		return true
	default:
		return false
	}
}

// GetAggregateSignatureScheme - get an aggregate signature scheme
func GetAggregateSignatureScheme(sigScheme string, total int, batchSize int) AggregateSignatureScheme {
	switch sigScheme {
	case SignatureSchemeEd25519:
		return nil
	case SignatureSchemeBls0chain:
		return NewBLS0ChainAggregateSignature(total, batchSize)
	default:
		panic(fmt.Sprintf("unknown signature scheme: %v", sigScheme))
	}
}

// IsValidThresholdSignatureScheme - whether a threshold signature scheme exists
func IsValidThresholdSignatureScheme(sigScheme string) bool {
	switch sigScheme {
	case SignatureSchemeEd25519:
		return false
	case SignatureSchemeBls0chain:
		return true
	default:
		return false
	}
}

// GetThresholdSignatureScheme - get a threshold signature scheme
func GetThresholdSignatureScheme(sigScheme string) ThresholdSignatureScheme {
	switch sigScheme {
	case SignatureSchemeEd25519:
		return nil
	case SignatureSchemeBls0chain:
		return NewBLS0ChainThresholdScheme()
	default:
		panic(fmt.Sprintf("unknown threshold signature scheme: %v", sigScheme))
	}
}

// GenerateThresholdKeyShares - generate T-of-N secret key shares for a key
func GenerateThresholdKeyShares(sigScheme string, t, n int, originalKey SignatureScheme) ([]ThresholdSignatureScheme, error) {
	switch sigScheme {
	case SignatureSchemeEd25519:
		return nil, nil
	case SignatureSchemeBls0chain:
		return BLS0GenerateThresholdKeyShares(t, n, originalKey)
	default:
		panic(fmt.Sprintf("unknown threshold signature scheme: %v", sigScheme))
	}
}

// IsValidReconstructSignatureScheme - whether a signature reconstruction scheme exists
func IsValidReconstructSignatureScheme(sigScheme string) bool {
	switch sigScheme {
	case SignatureSchemeEd25519:
		return false
	case SignatureSchemeBls0chain:
		return true
	default:
		return false
	}
}

// GetReconstructSignatureScheme - get a signature reconstruction scheme
func GetReconstructSignatureScheme(sigScheme string, t, n int) ReconstructSignatureScheme {
	switch sigScheme {
	case SignatureSchemeEd25519:
		return nil
	case SignatureSchemeBls0chain:
		return NewBLS0ChainReconstruction(t, n)
	default:
		panic(fmt.Sprintf("unknown signature reconstruction scheme: %v", sigScheme))
	}
}

// GetRawHash - given a hash interface (raw hash, hex string), return the raw hash
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

// VerifyPublicKeyClientID verifies if the clientID is generated from the public key
func VerifyPublicKeyClientID(pubKey string, clientID string) error {
	pubKeyBytes, err := hex.DecodeString(pubKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %v", err)
	}

	if Hash(pubKeyBytes) != clientID {
		return fmt.Errorf("mismatched public key and client ID")
	}

	return nil
}
