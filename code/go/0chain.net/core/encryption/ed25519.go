package encryption

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/ed25519"
)

// ED25519Scheme - a signature scheme based on ED25519
type ED25519Scheme struct {
	privateKey []byte
	publicKey  []byte
}

// NewED25519Scheme - create a ED255219Scheme object
func NewED25519Scheme() *ED25519Scheme {
	return &ED25519Scheme{}
}

// GenerateKeys - implement interface
func (ed *ED25519Scheme) GenerateKeys() error {
	public, private, err := GenerateKeysBytes()
	if err != nil {
		return err
	}
	ed.privateKey = private
	ed.publicKey = public
	return nil
}

// ReadKeys - implement interface
func (ed *ED25519Scheme) ReadKeys(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	result := scanner.Scan()
	if !result {
		return ErrKeyRead
	}
	publicKey := scanner.Text()
	if err := ed.SetPublicKey(publicKey); err != nil {
		return err
	}
	result = scanner.Scan()
	if !result {
		return ErrKeyRead
	}
	privateKey := scanner.Text()
	privateKeyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return err
	}
	ed.privateKey = privateKeyBytes
	return nil
}

// WriteKeys - implement interface
func (ed *ED25519Scheme) WriteKeys(writer io.Writer) error {
	publicKey := hex.EncodeToString(ed.publicKey)
	privateKey := hex.EncodeToString(ed.privateKey)
	_, err := fmt.Fprintf(writer, "%v\n%v\n", publicKey, privateKey)
	return err
}

// SetPublicKey - implement interface
func (ed *ED25519Scheme) SetPublicKey(publicKey string) error {
	if len(ed.privateKey) > 0 {
		return errors.New("cannot set public key when there is a private key")
	}
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return err
	}
	ed.publicKey = publicKeyBytes
	return nil
}

// GetPublicKey - implement interface
func (ed *ED25519Scheme) GetPublicKey() string {
	return hex.EncodeToString(ed.publicKey)
}

// Sign - impelemnt interface
func (ed *ED25519Scheme) Sign(hash interface{}) (string, error) {
	return signED25519(ed.privateKey, hash)
}

// Verify - implement interface
func (ed *ED25519Scheme) Verify(signature string, hash string) (bool, error) {
	return verifyED25519(ed.publicKey, signature, hash)
}

func signED25519(privateKey interface{}, hash interface{}) (string, error) {
	var pkBytes []byte
	switch pkImpl := privateKey.(type) {
	case []byte:
		pkBytes = pkImpl
	case string:
		decoded, err := hex.DecodeString(pkImpl)
		if err != nil {
			return "", err
		}
		pkBytes = decoded
	}
	rawHash, err := GetRawHash(hash)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ed25519.Sign(pkBytes, rawHash)), nil
}

func verifyED25519(publicKey interface{}, signature string, hash string) (bool, error) {
	var public []byte
	switch publicImpl := publicKey.(type) {
	case []byte:
		public = publicImpl
	case HashBytes:
		public = publicImpl[:]
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
