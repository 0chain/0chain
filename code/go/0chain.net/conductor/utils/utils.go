package utils

import (
	"0chain.net/core/encryption"
)

var signature = encryption.NewBLS0ChainScheme()

func init() {
	if err := signature.GenerateKeys(); err != nil {
		panic(err)
	}
}

// Sign by internal ("wrong") secret key generated randomly once client created.
func Sign(hash string) (sign string, err error) {
	return signature.Sign(hash)
}
