package encryption

import (
	"math/rand"
	"strconv"
	"testing"
)

func TestCreateMerkleTree(t *testing.T) {

	var transactionIDs = make([]string, 8)
	var transactionString string
	for i := 0; i < 8; i++ {
		t := strconv.Itoa(rand.Int())
		transactionIDs[i] = Hash(t)
		transactionString = transactionString + transactionIDs[i]
	}
	//fmt.Printf("The list of transactionIDs are : %v\n", transactionIDs)
	//fmt.Println(len(transactionIDs))

	MerkleTree, err := CreateMerkleTree(transactionIDs)
	if err != nil {
		t.Error("Unexpected error: ", err)
	}
	if MerkleTree.GetMerkleRoot() == Hash(transactionString) {
		t.Errorf("Error, expected hash equal to %v got %v", Hash(transactionString), MerkleTree.GetMerkleRoot())
	}

}

func TestCountLeafNodes(t *testing.T) {
	var transactionIDs = make([]string, 100)
	for i := 0; i < 100; i++ {
		t := strconv.Itoa(rand.Int())
		transactionIDs[i] = Hash(t)
	}
	//fmt.Printf("The list of transactionIDs are : %v\n", transactionIDs)
	//fmt.Println(len(transactionIDs))
	MerkleTree, err := CreateMerkleTree(transactionIDs)
	if err != nil {
		t.Error("Unexpected error :  ", err)
	}
	if len(MerkleTree.Leafs) != len(transactionIDs) {
		t.Errorf("The leaf counts do not match with the number of transactionIDs")
	}

}
