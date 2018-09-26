package encryption

import (
	"errors"
	"io"
)

var ErrKeyRead = errors.New("error reading the keys")

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
