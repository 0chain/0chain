package encryption

import (
	"fmt"
	"testing"

	"0chain.net/util"
)

type MerkleTestTransaction struct {
	TestTransactionData string
	TestTransactionID   string
}

/*GetHash function which has the hash */
func (t MerkleTestTransaction) GetHash() string {
	return t.TestTransactionID
}

func TestComputeTree(t *testing.T) {
	var transactions = make([]util.Hashable, 4)
	TesttransactionDataSample := []string{"a", "b", "c", "d"}
	for i := 0; i < 4; i++ {
		obj := new(MerkleTestTransaction)
		obj.TestTransactionData = TesttransactionDataSample[i]
		obj.TestTransactionID = Hash(obj.TestTransactionData)
		transactions[i] = obj
		//fmt.Printf("The transactionIDs are : %v\n", obj.TestTransactionID)
	}

	MerkleTree, err := ComputeTree(transactions)
	if err != nil {
		t.Error("Unexpected error: ", err)
	}
	fmt.Printf("The merkle root is : %v\n", MerkleTree.GetRoot())
	if MerkleTree.GetRoot() != "d6e2aad5041c946230d800699d0c0412bca22504049cc2e3559b8207912a8b1c" {
		t.Errorf("error: expected hash equal to d6e2aad5041c946230d800699d0c0412bca22504049cc2e3559b8207912a8b1c got %v", MerkleTree.GetRoot())
	}
}

func TestComputeTreeReverseOrder(t *testing.T) {
	var transactions = make([]util.Hashable, 2)
	TesttransactionDataSample := []string{"a", "b"}
	for i := 0; i < 2; i++ {
		obj := new(MerkleTestTransaction)
		obj.TestTransactionData = TesttransactionDataSample[i]
		obj.TestTransactionID = Hash(obj.TestTransactionData)
		transactions[i] = obj
		//fmt.Printf("The transactionIDs are : %v\n", obj.TestTransactionID)
	}
	MerkleTree, err := ComputeTree(transactions)
	if err != nil {
		t.Error("Unexpected error: ", err)
	}
	fmt.Printf("The merkle root is : %v\n", MerkleTree.GetRoot())
	if MerkleTree.GetRoot() == "92f024e8302bcda65b7ef274be43d0d4b23dc96db3211f212e8ae96705dadb0d" {
		t.Errorf("error: expected hash equal is %v", MerkleTree.GetRoot())
	}
}
