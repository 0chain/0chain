package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"0chain.net/common"
	"0chain.net/encryption"
)

func main() {
	keysFile := flag.String("keys_file", "keys.txt", "keys_file")
	data := flag.String("data", "", "data")
	timestamp := flag.Bool("timestamp", true, "timestamp")
	generateKeys := flag.Bool("generate_keys", false, "generate_keys")
	flag.Parse()
	if *generateKeys {
		publicKey, privateKey := encryption.GenerateKeys()
		if len(*keysFile) > 0 {
			data := []byte(fmt.Sprintf("%v\n%v", publicKey, privateKey))
			err := ioutil.WriteFile(*keysFile, data, 0600)
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Printf("%v\n", publicKey)
			fmt.Printf("%v\n", privateKey)
		}
	}
	if len(*keysFile) == 0 {
		return
	}
	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	publicKey, privateKey := encryption.ReadKeys(reader)
	clientID := encryption.Hash(publicKey)
	reader.Close()
	time := common.Now()
	fmt.Printf("data: %v\n", *data)
	fmt.Printf("keys file: %v\n", *keysFile)
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
	sign, err := encryption.Sign(privateKey, hash)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	} else {
		fmt.Printf("signature:%v\n", sign)
	}
}
