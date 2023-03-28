package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"0chain.net/core/common"
	"0chain.net/core/encryption"
)

func main() {
	clientSigScheme := flag.String("signature_scheme", "", "ed25519 or bls0chain")
	keysFileName := flag.String("keys_file_name", "keys.txt", "keys_file_name")
	path := flag.String("keys_file_path", "keys.txt", "keys_file_path")
	data := flag.String("data", "", "data")
	timestamp := flag.Bool("timestamp", true, "timestamp")
	generateKeys := flag.Bool("generate_keys", false, "generate_keys")
	flag.Parse()
	keysFile := fmt.Sprintf("%s/%s", *path, *keysFileName)
	var sigScheme = encryption.GetSignatureScheme(*clientSigScheme)
	if *generateKeys {
		err := sigScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
		if len(keysFile) > 0 {
			writer, err := os.OpenFile(keysFile, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				panic(err)
			}
			defer writer.Close()
			err = sigScheme.WriteKeys(writer)
			if err != nil {
				panic(err)
			}
		} else {
			err = sigScheme.WriteKeys(os.Stdout)
			if err != nil {
				panic(err)
			}
		}
	}
	if len(keysFile) == 0 {
		return
	}
	reader, err := os.Open(keysFile)
	if err != nil {
		panic(err)
	}
	_, publicKey, _ := encryption.ReadKeys(reader)
	pubKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		panic(err)
	}
	clientID := encryption.Hash(pubKeyBytes)
	reader.Close()
	time := common.Now()
	fmt.Printf("data: %v\n", *data)
	fmt.Printf("public_key: %v\n", publicKey)
	fmt.Printf("timestamp: %v\n", time)
	fmt.Printf("client_id: %v\n", clientID)
	var hashdata string
	if *timestamp {
		hashdata = fmt.Sprintf("%v:%v:%v\n", clientID, time, *data)
	} else {
		hashdata = fmt.Sprintf("%v:%v\n", clientID, *data)
	}
	fmt.Printf("hashdata: %v", hashdata)
	hash := encryption.Hash(hashdata)
	fmt.Printf("hash: %v\n", hash)
	sign, err := sigScheme.Sign(hash)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	} else {
		fmt.Printf("signature:%v\n", sign)
	}
}
