package util

import "0chain.net/encryption"

/*Hashable - anything that can provide it's hash */
type Hashable interface {
	GetHash() string
}

/*HashableBytes - a hashable object that returns the hash in bytes */
type HashableBytes interface {
	GetHashBytes() []byte
}

/*PersistedValue interface */
type PersistedValue interface {
	Encode() []byte
	Decode([]byte) error
}

/*SecurePersistedValue interface */
type SecurePersistedValue interface {
	PersistedValue
	HashableBytes
}

/*SecurePersistedValueImpl - a proxy persisted value that just tracks the encoded bytes of a persisted value */
type SecurePersistedValueImpl struct {
	Buffer []byte
}

/*GetHashBytes - implement interface */
func (spv *SecurePersistedValueImpl) GetHashBytes() []byte {
	return encryption.RawHash(spv.Buffer)
}

/*Encode - implement interface */
func (spv *SecurePersistedValueImpl) Encode() []byte {
	return spv.Buffer
}

/*Decode - implement interface */
func (spv *SecurePersistedValueImpl) Decode(buf []byte) error {
	spv.Buffer = buf
	return nil
}
