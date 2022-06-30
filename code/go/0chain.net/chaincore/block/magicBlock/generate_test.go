package main

import (
	"log"
	"testing"

	"github.com/herumi/bls/ffi/go/bls"
)

func TestKeyGeneration(t *testing.T) {
	hexString := "1234c153219f3688b8715670dd9d28d54e93f2c44bf65d0036c604a199a7a623"

	pk, pub := GenerateKeys()
	log.Println(pk, pub)
	var privateKey bls.SecretKey
	if err := privateKey.SetHexString(hexString); err != nil {
		log.Panic(err)
	}

}

//GenerateKeys - implement interface
func GenerateKeys() (string, string) {
	var skey bls.SecretKey
	skey.SetByCSPRNG()
	pub := skey.GetPublicKey().SerializeToHexStr()
	return skey.GetHexString(), pub
}
