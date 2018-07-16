package encryption

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

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
	case string:
		databuf = stringToBytes(dataImpl)
	default:
		panic("unknown type")
	}
	hash := sha3.New256()
	hash.Write(databuf)
	var buf []byte
	return hash.Sum(buf)
}

/*hexToString - convert either a regular string or hex string to byte array */
func stringToBytes(data string) []byte {
	var hdata []byte
	var err error
	if len(data)%2 == 1 {
		hdata, err = hex.DecodeString("0" + data)
	} else {
		hdata, err = hex.DecodeString(data)
	}
	if err == nil {
		return hdata
	} else {
		return []byte(data)
	}
}
