package main

import (
	"encoding/json"
	"fmt"

	sci "0chain.net/smartcontractinterface"
)

func main() {
	var smartContractData sci.SmartContractTransactionData
	byt := []byte(`{"name":"storage_test","input":"Test input to the miner"}`)
	err := json.Unmarshal(byt, &smartContractData)
	if err != nil {
		fmt.Println("Error in decoding JSON", err)
	} else {
		fmt.Println(smartContractData)
	}
}
