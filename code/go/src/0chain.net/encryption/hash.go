package encryption

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

/*Hash - Logic to hash the text*/
func Hash(text string) string {
	hash := sha3.New256()
	hash.Write([]byte(text))
	var buf []byte
	buf = hash.Sum(buf)
	return hex.EncodeToString(buf)
}
