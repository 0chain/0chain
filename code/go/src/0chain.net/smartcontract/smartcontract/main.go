package main

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontractstate"
)

type StorageNode struct {
	ID        string `json:"id"`
	BaseURL   string `json:"url"`
	PublicKey string `json:"-"`
}

func (sn *StorageNode) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *StorageNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	pndb, err := smartcontractstate.NewPSCDB("test_rocksdb")
	fmt.Println(err)
	var newBlobber StorageNode
	newBlobber.ID = "afc24d0e0e7a8afaaabc08bc49f5e415ab890ea1190d8281adf496e2960cd702"
	newBlobber.BaseURL = "http://localhost:5050"
	pndb.PutNode(smartcontractstate.Key(newBlobber.ID), newBlobber.Encode())

	pndb.Flush()

	var sn StorageNode
	node, err := pndb.GetNode(smartcontractstate.Key("afc24d0e0e7a8afaaabc08bc49f5e415ab890ea1190d8281adf496e2960cd702"))
	if err != nil || node == nil {
		fmt.Println(err)
		fmt.Println("Node not found in DB")
	}
	sn.Decode(node)
	fmt.Println(sn.ID)
	fmt.Println(sn.BaseURL)
}
