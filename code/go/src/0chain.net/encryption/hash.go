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
