package encryption

import (
	"math/rand"
	"strconv"
	"testing"
)

type MerkleTestTransaction struct {
	TestTransactionID   string
	TestTransactionData string
}

/*GetHashID - Entity implementation for Merkle tree */
func (t *MerkleTestTransaction) GetHashID() string {
	return t.TestTransactionID
}

func TestCountLeafNodesEven(t *testing.T) {
	var transactions = make([]MerkleTreeInterface, 20)
	for i := 0; i < 20; i++ {
		obj := new(MerkleTestTransaction)
		obj.TestTransactionData = strconv.Itoa(rand.Int())
		obj.TestTransactionID = Hash(obj.TestTransactionData)
		transactions[i] = obj
		//	fmt.Printf("The transactionIDs are : %v\n", obj.TestTransactionID)
	}
	MerkleTree, err := CreateMerkleTree(transactions)
	if err != nil {
		t.Error("Unexpected error: ", err)
	}
	if len(MerkleTree.Leafs) != len(transactions) {
		t.Errorf("The leaf counts do not match with the number of transactionIDs")
	}
}

func TestCountLeafNodesOdd(t *testing.T) {
	var transactions = make([]MerkleTreeInterface, 9)
	for i := 0; i < 9; i++ {
		obj := new(MerkleTestTransaction)
		obj.TestTransactionData = strconv.Itoa(rand.Int())
		obj.TestTransactionID = Hash(obj.TestTransactionData)
		transactions[i] = obj
		//fmt.Printf("The transactionIDs are : %v\n", obj.TestTransactionID)
	}
	MerkleTree, err := CreateMerkleTree(transactions)
	if err != nil {
		t.Error("Unexpected error: ", err)
	}
	if len(MerkleTree.Leafs) == len(transactions) {
		t.Errorf("The leaf counts do not match for odd number of transactionIDs")
	}
}
