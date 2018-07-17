package encryption

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

const HASH_LENGTH = 32

type HashBytes [HASH_LENGTH]byte

/*Hash - hash the given data and return the hash as hex string */
func Hash(data interface{}) string {
	return hex.EncodeToString(RawHash(data))
}

func (h *HashBytes) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HASH_LENGTH:]
	}
	copy(h[HASH_LENGTH-len(b):], b)
}

func (h *HashBytes) SetBytesFromString(s string) {
	b, _ := hex.DecodeString(s)
	if len(b) > len(h) {
		b = b[len(b)-HASH_LENGTH:]
	}
	copy(h[HASH_LENGTH-len(b):], b)
}

/*RawHash - Logic to hash the text and return the hash bytes */
func RawHash(data interface{}) []byte {
	var databuf []byte
	switch dataImpl := data.(type) {
	case []byte:
		databuf = dataImpl
	case HashBytes:
		databuf = dataImpl[:]
	case string:
		databuf = []byte(dataImpl)
	default:
		panic("unknown type")
	}
	hash := sha3.New256()
	hash.Write(databuf)
	var buf []byte
	return hash.Sum(buf)
}
