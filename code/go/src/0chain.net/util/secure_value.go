package util

import (
	"encoding/hex"

	"0chain.net/encryption"
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

func HashStringToBytes(hash string) []byte {
	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return nil
	}
	return hashBytes
}

/*SecureSerializableValueI an interface that makes a serializable value secure with hashing */
type SecureSerializableValueI interface {
	Serializable
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

/*GetHashBytes - implement interface */
func (spv *SecureSerializableValue) GetHashBytes() []byte {
	return encryption.RawHash(spv.Buffer)
}

/*Encode - implement interface */
func (spv *SecureSerializableValue) Encode() []byte {
	return spv.Buffer
}

/*Decode - implement interface */
func (spv *SecureSerializableValue) Decode(buf []byte) error {
	spv.Buffer = buf
	return nil
}
