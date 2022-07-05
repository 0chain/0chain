package util

import (
	"encoding/hex"
	"strings"

	"0chain.net/core/encryption"
	"github.com/tinylib/msgp/msgp"
)

/*Hashable - anything that can provide it's hash */
type Hashable interface {
	GetHash() string
	GetHashBytes() []byte
}

/*Serializable interface */
type Serializable interface {
	Encode() []byte
	Decode([]byte) error
}

//go:generate mockery --inpackage --case underscore --name MPTSerializable --testonly
// MPTSerializable represents the interface for encoding/decoding
// data that stores in MPT
type MPTSerializable interface {
	msgp.Marshaler
	msgp.Unmarshaler
}

// MPTSerializableSize wraps the MPTSerializable and msgp.Sizer interfaces
type MPTSerializableSize interface {
	MPTSerializable
	msgp.Sizer
}

/*HashStringToBytes - convert a hex hash string to bytes */
func HashStringToBytes(hash string) []byte {
	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return nil
	}
	return hashBytes
}

/*SecureSerializableValueI an interface that makes a serializable value secure with hashing */
type SecureSerializableValueI interface {
	MPTSerializable
	Hashable
}

/*SecureSerializableValue - a proxy persisted value that just tracks the encoded bytes of a persisted value */
type SecureSerializableValue struct {
	Buffer []byte
}

/*GetHash - implement interface */
func (spv *SecureSerializableValue) GetHash() string {
	return ToHex(spv.GetHashBytes())
}

/*ToHex - converts a byte array to hex encoding */
func ToHex(buf []byte) string {
	return hex.EncodeToString(buf)
}

func fromHex(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

/*ToUpperHex - converts a byte array to hex encoding with upper case */
func ToUpperHex(buf []byte) string {
	return strings.ToUpper(hex.EncodeToString(buf))
}

/*GetHashBytes - implement interface */
func (spv *SecureSerializableValue) GetHashBytes() []byte {
	return encryption.RawHash(spv.Buffer)
}

// MarshalMsg encodes node and implement msg.Marshaler interface
func (spv *SecureSerializableValue) MarshalMsg([]byte) ([]byte, error) {
	return spv.Buffer, nil
}

// UnmarshalMsg decodes node and implement msgp.Unmarshaler interface
func (spv *SecureSerializableValue) UnmarshalMsg(buf []byte) ([]byte, error) {
	spv.Buffer = buf
	return nil, nil
}
